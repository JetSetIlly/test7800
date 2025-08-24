package debugger

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"github.com/jetsetilly/dialog"
	"github.com/jetsetilly/test7800/disassembly"
	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware"
	"github.com/jetsetilly/test7800/hardware/arm"
	"github.com/jetsetilly/test7800/hardware/cpu"
	"github.com/jetsetilly/test7800/hardware/cpu/execution"
	"github.com/jetsetilly/test7800/hardware/maria"
	"github.com/jetsetilly/test7800/hardware/memory/external"
	"github.com/jetsetilly/test7800/logger"
	"github.com/jetsetilly/test7800/resources"
)

type input struct {
	s   string
	err error
}

type debugger struct {
	ctx context

	guiQuit chan bool
	sig     chan os.Signal
	input   chan input

	// this channel is poassed to the debugger during creation via the UI type
	state chan gui.State

	console        *hardware.Console
	breakpoints    map[uint16]bool
	watches        map[uint16]watch
	breakspointCtx bool

	// recent execution results to be printed on emulation halt
	recent []execution.Result

	// coprocessor disassembly and development environments
	coprocDisasm *coprocDisasm
	coprocDev    *coprocDev

	// rule for stepping. by default (the field is nil) the step will move
	// forward one instruction
	stepRule func() bool

	// the file to load on console reset. can be a bootfile or cartridge
	loader string

	// script of commands
	script []string

	// printing styles
	styles styles

	// some cartridge types will bypass the BIOS. it's possible to force the
	// BIOS to be skipped with this flag in all cases
	bypassBIOS bool
}

func (m *debugger) reset() {
	m.ctx.Reset()

	var cartridgeReset external.CartridgeReset

	// load file specified by loader
	if m.loader != "" {
		c, err := external.Fingerprint(m.loader)
		if err != nil {
			if errors.Is(err, external.UnrecognisedData) {
				// file is not a cartridge dump so we'll assume it's a bootfile
				fmt.Println(m.styles.debugger.Render(
					fmt.Sprintf("booting from %s", filepath.Base(m.loader)),
				))

				m.script, err = m.bootFromFile(c.Data())
				if err == nil {
					// resetting with a boot file is a bit different because we
					// don't want to do a normal reset if the boot process was
					// succesful
					return
				}

				// forget about loader because we now know it doesn't work
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("%s: %s", filepath.Base(m.loader), err.Error()),
				))
				m.loader = ""
			} else {
				// forget about loader because we now know it doesn't work
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("%s: %s", filepath.Base(m.loader), err.Error()),
				))
				m.loader = ""
			}

		} else {
			err = m.console.Insert(c)
			if err != nil {
				fmt.Println(m.styles.err.Render(err.Error()))
			} else {
				fmt.Println(m.styles.debugger.Render(
					fmt.Sprintf("%s cartridge from %s", m.console.Mem.External.Label(),
						filepath.Base(m.loader)),
				))
				cartridgeReset = c.ResetProcedure()
			}
		}
	}

	// try and (re)attach coproc developer/disassembly to external device
	coproc := m.console.Mem.External.GetCoProcHandler()
	if coproc != nil {
		coproc.GetCoProc().SetDeveloper(m.coprocDev)
		if m.coprocDisasm.enabled {
			coproc.GetCoProc().SetDisassembler(m.coprocDisasm)
		}
		coproc.SetYieldHook(m)
	}

	var noBIOS = m.bypassBIOS || cartridgeReset.BypassBIOS

	err := m.console.Reset(true)
	if err != nil {
		fmt.Println(m.styles.err.Render(err.Error()))
	} else {
		if noBIOS {
			fmt.Println(m.styles.debugger.Render("console reset with no BIOS"))
		} else {
			fmt.Println(m.styles.debugger.Render("console reset"))
		}
	}

	if noBIOS {
		// writing to the INPTCTRL twice to make sure the halt line has been enabled
		m.console.Mem.INPTCTRL.Write(0x01, 0x07)
		m.console.Mem.INPTCTRL.Write(0x01, 0x07)

		// set 6507 program-counter to normal reset address
		m.console.MC.LoadPCIndirect(cpu.Reset)
		if err != nil {
			fmt.Println(m.styles.err.Render(err.Error()))
		}

		// feedback on the current state of INPTCTRL
		fmt.Println(m.styles.cpu.Render(
			m.console.Mem.INPTCTRL.Status(),
		))
	}

	fmt.Println(m.styles.mem.Render(
		m.console.Mem.BIOS.Status(),
	))
	fmt.Println(m.styles.cpu.Render(
		m.console.MC.String(),
	))
}

func (m *debugger) contextBreaks() error {
	if len(m.ctx.Breaks) == 0 {
		return nil
	}

	if !m.breakspointCtx {
		m.ctx.Breaks = m.ctx.Breaks[:0]
		return nil
	}

	// filter errors to only deal with the ones we're interested in
	// TODO: configurable filters
	var f []error
	for _, e := range m.ctx.Breaks {
		if !errors.Is(e, maria.ContextError) {
			f = append(f, e)
		}
	}

	// breaks have been processed and so are now cleared
	m.ctx.Breaks = m.ctx.Breaks[:0]

	if len(f) == 0 {
		return nil
	}

	// concatenate filtered errors for possible display
	err := f[0]
	for _, e := range f[1:] {
		err = fmt.Errorf("%w\n%w", err, e)
	}

	return err
}

func (m *debugger) printInstruction(res *disassembly.Entry) {
	if res.Result.InInterrupt {
		fmt.Print(m.styles.instruction.Render("!! "))
	}
	fmt.Println(m.styles.instruction.Render(
		strings.TrimSpace(fmt.Sprintf("%s %s %s", res.Address, res.Operator, res.Operand))),
	)
}

func (m *debugger) last() {
	res := disassembly.FormatResult(m.console.MC.LastResult)
	m.printInstruction(res)
}

// the number of recent instructions to record. also used to clip the number of
// coproc instructions to output on error
const maxRecentLen = 100

// returns true if quit signal has been received from the GUI
func (m *debugger) run() bool {
	if m.stepRule == nil {
		fmt.Println(m.styles.debugger.Render("emulation running"))
	}

	// we measure the number of instructions in the time period of the running emulation
	var instructionCt int
	var startTime time.Time

	// sentinal errors to
	var (
		coprocErr     = errors.New("coproc")
		breakpointErr = errors.New("breakpoint")
		watchErr      = errors.New("watch")
		contextErr    = errors.New("context")
		endRunErr     = errors.New("end run")
		quitErr       = errors.New("quit")
	)

	// always cancel stepping rule
	defer func() {
		m.stepRule = nil
	}()

	// hook is called after every CPU instruction
	hook := func() error {
		select {
		case <-m.sig:
			return endRunErr
		case <-m.guiQuit:
			return quitErr
		default:
		}

		// record last instruction
		if m.console.MC.LastResult.Final {
			m.recent = append(m.recent, m.console.MC.LastResult)
			if len(m.recent) > maxRecentLen {
				m.recent = m.recent[1:]
			}
		}

		instructionCt++

		if m.console.MC.Killed {
			return fmt.Errorf("CPU in KIL state")
		}

		if m.coprocDev != nil {
			if len(m.coprocDev.faults.Log) > 0 {
				return fmt.Errorf("%w%s", coprocErr, m.coprocDev.faults.Log[len(m.coprocDev.faults.Log)-1].String())
			}
		}

		err := m.contextBreaks()
		if err != nil {
			return fmt.Errorf("%w%w", contextErr, err)
		}

		pcAddr := m.console.MC.PC.Address()
		if _, ok := m.breakpoints[pcAddr]; ok {
			return fmt.Errorf("%w: %04x", breakpointErr, pcAddr)
		}

		w, err := m.checkWatches()
		if err != nil {
			return fmt.Errorf("%w%w", contextErr, err)
		}
		if w != nil {
			return fmt.Errorf("%w: %04x = %02x -> %02x", watchErr, w.ma.address, w.prev, w.data)
		}

		// apply step rule and end the run if instructed
		if m.stepRule != nil && m.stepRule() {
			return endRunErr
		}

		// swallow last area status before next iteration. doing this here means that
		// the last area status will not printed when the run ends unless it was the
		// affected by the most recent instruction
		_ = m.console.LastAreaStatus()

		return nil
	}

	startTime = time.Now()

	m.state <- gui.StateRunning
	err := m.console.Run(hook)
	m.state <- gui.StatePaused

	if errors.Is(err, quitErr) {
		return true
	}

	m.console.MARIA.PushRender()

	if m.stepRule == nil {
		// output recent CPU instructons on end of a non-step run
		if len(m.recent) > 1 {
			fmt.Println(m.styles.debugger.Render("most recent CPU instructions"))
			n := max(len(m.recent)-10, 0)
			for _, e := range m.recent[n:] {
				res := disassembly.FormatResult(e)
				m.printInstruction(res)
			}
		}
		fmt.Println(m.styles.cpu.Render(
			m.console.MC.String(),
		))
	} else {
		m.last()
		fmt.Println(m.styles.cpu.Render(
			m.console.MC.String(),
		))
		if s := m.console.LastAreaStatus(); len(s) > 0 {
			fmt.Println(m.styles.mem.Render(s))
		}
	}

	// output most recent coproc disassembly if enabled. we call this in the
	// event of a coprocErr
	if m.coprocDisasm.enabled {
		n := max(0, len(m.coprocDisasm.last)-10)
		for _, e := range m.coprocDisasm.last[n:] {
			// print processor specific information as appropriate
			if a, ok := e.(arm.DisasmEntry); ok {
				bytecode := fmt.Sprintf("%04x", a.Opcode)
				if a.Is32bit {
					bytecode = fmt.Sprintf("%04x %s", a.OpcodeHi, bytecode)
				} else {
					bytecode = fmt.Sprintf("%s     ", bytecode)
				}

				var annotation string
				if a.Annotation != nil {
					annotation = fmt.Sprintf("\t\t(%s)", a.Annotation.String())
				}
				fmt.Println(m.styles.coprocAsm.Render(
					fmt.Sprintf("%s %s %s%s", a.Address, bytecode, a.String(), annotation),
				))
			} else {
				fmt.Println(m.styles.coprocAsm.Render(
					fmt.Sprintf("%s %s", e.Key(), e.String()),
				))
			}
		}
	}

	// instruction count and time elapsed
	if m.stepRule == nil || instructionCt > 1 {
		fmt.Println(m.styles.debugger.Render(
			fmt.Sprintf("%d instructions in %.02f seconds", instructionCt, time.Since(startTime).Seconds())),
		)
	}

	if errors.Is(err, endRunErr) {
		// nothing else to do in the case of an endRunErr error
	} else if errors.Is(err, coprocErr) {
		s := strings.TrimPrefix(err.Error(), coprocErr.Error())
		fmt.Println(m.styles.coprocErr.Render(s))
	} else if errors.Is(err, breakpointErr) {
		fmt.Println(m.styles.breakpoint.Render(err.Error()))
	} else if errors.Is(err, watchErr) {
		fmt.Println(m.styles.watch.Render(err.Error()))
	} else if errors.Is(err, contextErr) {
		s := strings.TrimPrefix(err.Error(), contextErr.Error())
		fmt.Println(m.styles.err.Render(s))
	} else if err != nil {
		fmt.Println(m.styles.err.Render(err.Error()))
	}

	return false
}

func (m *debugger) loop() {
	for {
		fmt.Printf("%s> ", m.console.MARIA.Coords.ShortString())

		var cmd []string

		select {
		case input := <-m.input:
			if input.err != nil {
				fmt.Println(m.styles.err.Render(input.err.Error()))
				return
			}
			cmd = strings.Fields(input.s)
			if len(cmd) == 0 {
				cmd = []string{"STEP"}
			}
		case <-m.sig:
			fmt.Print("\r")
			return
		case <-m.guiQuit:
			fmt.Print("\n")
			return
		}

		if m.commands(cmd) {
			return
		}
	}
}

const programName = "test7800"

func Launch(guiQuit chan bool, g *gui.GUI, args []string) error {
	var filename string
	var spec string
	var profile bool
	var bios bool
	var overlay bool
	var run bool
	var log bool
	var audio bool

	flgs := flag.NewFlagSet(programName, flag.ExitOnError)
	flgs.StringVar(&spec, "spec", "NTSC", "TV specification of the console: NTSC or PAL")
	flgs.BoolVar(&profile, "profile", false, "create CPU profile for emulator")
	flgs.BoolVar(&bios, "bios", true, "run BIOS routines on reset")
	flgs.BoolVar(&overlay, "overlay", false, "add debugging overlay to display")
	flgs.BoolVar(&run, "run", false, "start ROM in running state")
	flgs.BoolVar(&log, "log", false, "echo log to stderr")
	flgs.BoolVar(&audio, "audio", true, "enable audio")
	err := flgs.Parse(args)
	if err != nil {
		return err
	}
	args = flgs.Args()

	// exit program immediately if program launched with a file dialog
	var runQuitImmediately bool

	// if no filename has been specified then open a file dialog
	if len(args) == 0 {
		lastSelectedROM, err := resources.Read("lastSelectedROM")
		if err != nil {
			return err
		}

		dlg := dialog.File()
		dlg = dlg.Title("Select 7800 ROM")
		dlg = dlg.Filter("7800 Files", "a78", "bin", "elf", "boot")
		dlg = dlg.Filter("A78 Files Only", "a78")
		dlg = dlg.Filter("All Files")
		dlg = dlg.SetStartDir(filepath.Dir(lastSelectedROM))
		filename, err = dlg.Load()
		if err != nil {
			if errors.Is(err, dialog.ErrCancelled) {
				return nil
			}
			return err
		}

		_, err = external.Fingerprint(filename)
		if err != nil {
			dialog.Message("Problem with selected file\n\n%s", err.Error()).Info()
			return err
		}

		err = resources.Write("lastSelectedROM", filename)
		if err != nil {
			return err
		}

		// we always want to run immediately if the filename has been chosen through the file dialog
		run = true
		runQuitImmediately = true

	} else if len(args) == 1 {
		if args[0] != "-" {
			filename = args[0]
		}

	} else if len(args) > 1 {
		return fmt.Errorf("too many arguments to debugger")
	}

	if log {
		logger.SetEcho(os.Stderr, false)
	}

	ctx := context{
		console:    "7800",
		spec:       strings.ToUpper(spec),
		useOverlay: overlay,
		useAudio:   audio,
	}
	ctx.Reset()

	m := &debugger{
		ctx:          ctx,
		guiQuit:      guiQuit,
		state:        g.State,
		sig:          make(chan os.Signal, 1),
		input:        make(chan input, 1),
		loader:       filename,
		styles:       newStyles(),
		breakpoints:  make(map[uint16]bool),
		watches:      make(map[uint16]watch),
		coprocDisasm: &coprocDisasm{},
		coprocDev:    newCoprocDev(),
		bypassBIOS:   !bios,
	}
	m.console = hardware.Create(&m.ctx, g)
	m.console.Reset(true)

	signal.Notify(m.sig, syscall.SIGINT)

	go func() {
		r := bufio.NewReader(os.Stdin)
		b := make([]byte, 256)
		for {
			n, err := r.Read(b)
			select {
			case m.input <- input{
				s:   strings.TrimSpace(string(b[:n])),
				err: err,
			}:
			default:
			}
		}
	}()

	m.reset()

	if profile {
		f, err := os.Create("cpu.profile")
		if err != nil {
			return fmt.Errorf("performance: %w", err)
		}
		defer func() {
			err := f.Close()
			if err != nil {
				logger.Log(logger.Allow, "performance", err)
			}
		}()

		err = pprof.StartCPUProfile(f)
		if err != nil {
			return fmt.Errorf("performance: %w", err)
		}
		defer pprof.StopCPUProfile()
	}

	// start off gui in the paused state. gui won't properly begin until it receives a state change
	g.State <- gui.StatePaused

	// start in run state if required
	if run {
		if m.run() {
			return nil
		}
		if runQuitImmediately {
			return nil
		}
	}

	// start debugging loop
	m.loop()

	return nil
}
