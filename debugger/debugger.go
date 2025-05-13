package debugger

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jetsetilly/test7800/disassembly"
	"github.com/jetsetilly/test7800/hardware"
	"github.com/jetsetilly/test7800/hardware/arm"
	"github.com/jetsetilly/test7800/hardware/cpu"
	"github.com/jetsetilly/test7800/hardware/cpu/execution"
	"github.com/jetsetilly/test7800/hardware/maria"
	"github.com/jetsetilly/test7800/hardware/memory"
	"github.com/jetsetilly/test7800/hardware/memory/external"
	"github.com/jetsetilly/test7800/logger"
	"github.com/jetsetilly/test7800/ui"
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
	state chan ui.State

	console     hardware.Console
	breakpoints map[uint16]bool
	watches     map[uint16]watch

	// recent execution results to be printed on emulation halt
	recent []execution.Result

	// coprocessor disassembly and development environments
	coprocDisasm *coprocDisasm
	coprocDev    *coprocDev

	// rule for stepping. by default (the field is nil) the step will move
	// forward one instruction
	stepRule func() bool
	postStep func()

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
		d, err := ioutil.ReadFile(m.loader)
		if err != nil {
			fmt.Println(m.styles.err.Render(
				fmt.Sprintf("error loading %s: %s", m.loader, err.Error()),
			))
		} else {
			c, err := external.Fingerprint(d)
			if err != nil {
				if errors.Is(err, external.UnrecognisedData) {
					// file is not a cartridge dump so we'll assume it's a bootfile
					fmt.Println(m.styles.debugger.Render(
						fmt.Sprintf("booting from %s", filepath.Base(m.loader)),
					))

					m.script, err = m.bootFromFile(d)
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
				err = m.console.Mem.External.Insert(c)
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

	err := m.console.Reset(true)
	if err != nil {
		fmt.Println(m.styles.err.Render(err.Error()))
	} else {
		fmt.Println(m.styles.debugger.Render("console reset"))
	}

	if m.bypassBIOS || cartridgeReset.BypassBIOS {
		// writing to the INPTCTRL twice to make sure the halt line has been enabled
		m.console.Mem.INPTCTRL.Write(0x01, 0x07)
		m.console.Mem.INPTCTRL.Write(0x01, 0x07)

		// set 6507 program-counter to normal reset address
		m.console.MC.LoadPCIndirect(cpu.Reset)

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

// step advances the emulation on CPU instruction according to the current step
// the step rule will be reset after the step has completed
//
// returns true if quit signal has been received
func (m *debugger) step() bool {
	// the number of instructions stepped over
	var ct int

	// loop until the step rule returns true
	var done bool
	for !done {
		select {
		case <-m.sig:
			done = true
			continue // for loop
		case <-m.guiQuit:
			return true
		default:
		}

		err := m.console.Step()
		if err != nil {
			fmt.Println(m.styles.err.Render(
				err.Error(),
			))
			return false
		}

		// record last instruction
		if m.console.MC.LastResult.Final {
			m.recent = append(m.recent, m.console.MC.LastResult)
			if len(m.recent) > maxRecentLen {
				m.recent = m.recent[1:]
			}
		}

		if m.coprocDev != nil {
			if len(m.coprocDev.faults.Log) > 0 {
				fmt.Println(m.styles.coprocErr.Render(
					m.coprocDev.faults.Log[len(m.coprocDev.faults.Log)-1].String(),
				))
			}
		}

		err = m.contextBreaks()
		if err != nil {
			fmt.Println(m.styles.breakpoint.Render(err.Error()))
			return false
		}

		// apply step rule
		if m.stepRule == nil {
			done = true
		} else {
			done = m.stepRule()
		}

		ct++
	}

	m.console.MARIA.PushRender()

	// report how many instructions were stepped if it is more than one
	if ct > 1 {
		fmt.Println(m.styles.debugger.Render(
			fmt.Sprintf("%d instructions stepped", ct),
		))
	}

	if m.postStep == nil {
		// by default we print the general status of the emulation
		m.last()
		fmt.Println(m.styles.cpu.Render(
			m.console.MC.String(),
		))
		if s := m.console.LastAreaStatus(); len(s) > 0 {
			fmt.Println(m.styles.mem.Render(s))
		}
	} else {
		m.postStep()
	}

	m.stepRule = nil
	m.postStep = nil

	return false
}

func (m *debugger) printInstruction(res *disassembly.Entry) {
	if m.console.MC.InInterrupt() {
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

// returns true if quit signal has been received
func (m *debugger) run() bool {
	fmt.Println(m.styles.debugger.Render("emulation running"))

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

		return nil
	}

	startTime = time.Now()

	m.state <- ui.StateRunning
	err := m.console.Run(hook)
	m.state <- ui.StatePaused

	if errors.Is(err, quitErr) {
		return true
	}

	m.console.MARIA.PushRender()

	// output recent CPU instructons on end
	if len(m.recent) > 0 {
		fmt.Println(m.styles.debugger.Render("most recent CPU instructions"))
		n := max(len(m.recent)-10, 0)
		for _, e := range m.recent[n:] {
			res := disassembly.FormatResult(e)
			m.printInstruction(res)
		}
	}

	// output most recent coproc disassembly if enabled. we call this in the
	// event of a coprocErr
	outputCoprocDisasm := func() {
		if m.coprocDisasm.enabled {
			n := max(0, len(m.coprocDisasm.last)-maxRecentLen)
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
	}

	if errors.Is(err, endRunErr) {
		fmt.Println(m.styles.debugger.Render(
			fmt.Sprintf("%d instructions in %.02f seconds", instructionCt, time.Since(startTime).Seconds())),
		)
	} else if errors.Is(err, coprocErr) {
		outputCoprocDisasm()
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

	// it's useful to see the state of the CPU and the MARIA coords at the end of the run
	fmt.Println(m.styles.cpu.Render(m.console.MC.String()))
	fmt.Println(m.styles.video.Render(m.console.MARIA.Coords.String()))

	// consume last memory access information
	_ = m.console.LastAreaStatus()

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

		switch strings.ToUpper(cmd[0]) {
		case "BOOT":
			if len(cmd) < 5 {
				fmt.Println(m.styles.err.Render(
					"BOOT requires a ROM file, an origin address, an entry address and the INPTCTRL value",
				))
				break // switch
			}

			err := m.bootParse(cmd[1:])
			if err != nil {
				fmt.Println(m.styles.err.Render(err.Error()))
				break // switch
			}
		case "R", "RUN":
			if m.run() {
				return
			}
		case "ST", "STEP":
			if len(cmd) > 1 {
				if !m.parseStepRule(cmd[1:]) {
					break // switch
				}
			}
			if m.step() {
				return
			}
		case "RESET":
			m.reset()
		case "CPU":
			fmt.Println(m.styles.cpu.Render(
				m.console.MC.String(),
			))
		case "RECENT":
			n := 10
			if len(cmd) == 2 {
				var err error
				n, err = strconv.Atoi(cmd[1])
				if err != nil {
					fmt.Println(m.styles.err.Render(
						"cannot use RECENT %s", cmd[1],
					))
					break // switch
				}
			}
			n = max(len(m.recent)-n, 0)
			for _, e := range m.recent[n:] {
				res := disassembly.FormatResult(e)
				m.printInstruction(res)
			}

		case "BIOS":
			fmt.Println(m.styles.mem.Render(
				m.console.Mem.BIOS.Status(),
			))
		case "MARIA":
			fmt.Println(m.styles.mem.Render(
				m.console.MARIA.Status(),
			))
		case "DL":
			fmt.Println(m.styles.mem.Render(
				m.console.MARIA.DL.Status(),
			))
		case "DLL":
			if len(cmd) == 2 {
				if strings.ToUpper(cmd[1]) == "LIST" {
					for _, dll := range m.console.MARIA.RecentDLL {
						fmt.Println("")
						fmt.Println(m.styles.mem.Render(
							dll.String(),
						))
					}
				} else {
					fmt.Println(m.styles.err.Render(
						fmt.Sprintf("unrecognised argument for COPROC command: %s", cmd[2]),
					))
				}
			} else {
				fmt.Println(m.styles.mem.Render(
					m.console.MARIA.DLL.Status(),
				))
			}
		case "VIDEO":
			fmt.Println(m.styles.video.Render(
				m.console.MARIA.Coords.String(),
			))
		case "INPTCTRL":
			fmt.Println(m.styles.mem.Render(
				m.console.Mem.INPTCTRL.Status(),
			))
		case "RAM7800":
			fmt.Println(m.styles.mem.Render(
				m.console.Mem.RAM7800.String(),
			))
		case "RAMRIOT":
			fmt.Println(m.styles.mem.Render(
				m.console.Mem.RAMRIOT.String(),
			))
		case "DUMP":
			if len(cmd) < 3 {
				fmt.Println(m.styles.err.Render(
					"DUMP requires a 'from' and a 'to' address",
				))
				break // switch
			}

			from, err := m.parseAddress(cmd[1])
			if err != nil {
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("dump: %s", err.Error()),
				))
				break // switch
			}

			to, err := m.parseAddress(cmd[2])
			if err != nil {
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("dump: %s", err.Error()),
				))
				break // switch
			}

			if to.address < from.address {
				fmt.Println(m.styles.err.Render(
					"dump: the 'to' address is less than the 'from' address",
				))
				break // switch
			}

			if from.area != to.area {
				fmt.Println(m.styles.err.Render(
					"dump: the 'from' and 'to' addresses are in different memory areas",
				))
				break // switch
			}

			var column int
			for i := from.idx; i <= to.idx; i++ {
				address := from.address + i - from.idx

				if column == 0 {
					fmt.Printf("%04x", address)
				}

				data, err := memory.Read(from.area, i)
				if err != nil {
					fmt.Println(m.styles.err.Render(
						fmt.Sprintf("dump address is not readable: %04x", address),
					))
					break // switch
				}
				fmt.Printf(" %02x", data)

				column++
				if column > 15 {
					fmt.Printf("\n")
					column = 0
				}
			}
			if column != 0 {
				fmt.Printf("\n")
			}
		case "PEEK":
			if len(cmd) < 2 {
				fmt.Println(m.styles.err.Render(
					"PEEK requires an address",
				))
				break // switch
			}

			ma, err := m.parseAddress(cmd[1])
			if err != nil {
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("peek: %s", err.Error()),
				))
				break // switch
			}

			data, err := memory.Read(ma.area, ma.idx)
			if err != nil {
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("peek address is not readable: %s", cmd[1]),
				))
				break // switch
			}

			fmt.Println(m.styles.mem.Render(
				fmt.Sprintf("$%04x = %02x (%s)", ma.address, data, ma.area.Label()),
			))
		case "BREAK":
			if len(cmd) < 2 {
				fmt.Println(m.styles.err.Render(
					"BREAK requires an address",
				))
				break // switch
			}

			// we check the first argument for special keywords before assuming
			// it is an address. the keywords are case insensitive
			arg := strings.ToUpper(cmd[1])

			if arg == "DROP" {
				if len(cmd) < 3 {
					fmt.Println(m.styles.err.Render(
						"BREAK DROP requires an address",
					))
					break // switch
				}

				if strings.ToUpper(cmd[2]) == "ALL" {
					clear(m.breakpoints)
				} else {
					ma, err := m.parseAddress(cmd[2])
					if err != nil {
						fmt.Println(m.styles.err.Render(
							fmt.Sprintf("breakpoint: %s", err.Error()),
						))
						break // switch
					}
					if _, ok := m.breakpoints[ma.address]; !ok {
						fmt.Println(m.styles.debugger.Render(
							fmt.Sprintf("breakpoint for $%04x not present", ma.address),
						))
						break // switch
					}
					delete(m.breakpoints, ma.address)
					fmt.Println(m.styles.debugger.Render(
						fmt.Sprintf("breakpoint %04x has been removed", ma.address),
					))
				}
				break // switch
			}

			// the NEXT argument to BREAK is useful for setting a breakpoint on the instruction
			// on a failed branch instruction, which is a common action when stepping through
			// a program
			//
			// a STEP OVER command would be just as good but we don't have that at the moment
			if arg == "NEXT" {
				address := m.console.MC.LastResult.Address
				address += uint16(m.console.MC.LastResult.ByteCount)
				cmd[1] = fmt.Sprintf("%#04x", address)
			}

			ma, err := m.parseAddress(cmd[1])
			if err != nil {
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("breakpoint: %s", err.Error()),
				))
				break // switch
			}

			if _, ok := m.breakpoints[ma.address]; ok {
				fmt.Println(m.styles.debugger.Render(
					fmt.Sprintf("breakpoint on $%04x already present", ma.address),
				))
				break // switch
			}

			m.breakpoints[ma.address] = true
			fmt.Println(m.styles.debugger.Render(
				fmt.Sprintf("added breakpoint for $%04x", ma.address),
			))
		case "WATCH":
			if len(cmd) < 2 {
				fmt.Println(m.styles.err.Render(
					"WATCH requires an address",
				))
				break // switch
			}

			// we check the first argument for special keywords before assuming
			// it is an address. the keywords are case insensitive
			arg := strings.ToUpper(cmd[1])

			if arg == "DROP" {
				if len(cmd) < 3 {
					fmt.Println(m.styles.err.Render(
						"WATCH DROP requires an address",
					))
					break // switch
				}

				if strings.ToUpper(cmd[2]) == "ALL" {
					clear(m.watches)
				} else {
					ma, err := m.parseAddress(cmd[2])
					if err != nil {
						fmt.Println(m.styles.err.Render(
							fmt.Sprintf("watch: %s", err.Error()),
						))
						break // switch
					}
					if _, ok := m.watches[ma.address]; !ok {
						fmt.Println(m.styles.debugger.Render(
							fmt.Sprintf("watch for $%04x not present", ma.address),
						))
						break // switch
					}
					delete(m.watches, ma.address)
					fmt.Println(m.styles.debugger.Render(
						fmt.Sprintf("watch %04x has been removed", ma.address),
					))
				}
				break // switch
			}

			ma, err := m.parseAddress(cmd[1])
			if err != nil {
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("watch: %s", err.Error()),
				))
				break // switch
			}

			if _, ok := m.watches[ma.address]; ok {
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("watch for %s already present", cmd[1]),
				))
				break // switch
			}

			d, err := memory.Read(ma.area, ma.idx)
			if err != nil {
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("watch address is not readable: %s", cmd[1]),
				))
				break // switch
			}

			m.watches[ma.address] = watch{
				ma:   ma,
				data: d,
			}
		case "LIST":
			fmt.Println(m.styles.debugger.Render("breakpoints"))
			if len(m.breakpoints) == 0 {
				fmt.Println("none")
			} else {
				for a := range m.breakpoints {
					fmt.Printf("%#04x\n", a)
				}
			}
			fmt.Println(m.styles.debugger.Render("watches"))
			if len(m.watches) == 0 {
				fmt.Println("none")
			} else {
				for a := range m.watches {
					fmt.Printf("%#04x\n", a)
				}
			}
		case "COPROC":
			coproc := m.console.Mem.External.GetCoProcHandler()
			if coproc == nil {
				fmt.Println(m.styles.err.Render(
					"external device does not have a coprocessor",
				))
				break // switch
			}
			switch len(cmd) {
			case 1:
				fmt.Println(m.styles.debugger.Render(
					coproc.GetCoProc().ProcessorID(),
				))
			case 2:
				c := strings.ToUpper(cmd[1])
				switch c {
				case "DISASM":
					coproc.GetCoProc().SetDisassembler(m.coprocDisasm)
					m.coprocDisasm.enabled = true
				case "END":
					coproc.GetCoProc().SetDisassembler(nil)
					m.coprocDisasm.enabled = false
				case "FAULTS":
					if m.coprocDev != nil {
						if len(m.coprocDev.faults.Log) == 0 {
							fmt.Println(m.styles.debugger.Render(
								"no coprocessor memory faults",
							))
						} else {
							for _, f := range m.coprocDev.faults.Log {
								fmt.Println(f)
							}
						}
					}
				case "REGS", "REG":
					if s, ok := coproc.GetCoProc().(fmt.Stringer); ok {
						fmt.Println(m.styles.coprocCPU.Render(
							s.String(),
						))
					} else {
						fmt.Println(m.styles.coprocErr.Render(
							"no register information",
						))
					}
				default:
					fmt.Println(m.styles.err.Render(
						fmt.Sprintf("unrecognised argument for COPROC command: %s", c),
					))
				}
			default:
				fmt.Println(m.styles.err.Render(
					"too many arguments to COPROC command",
				))
			}
		case "LOG":
			logger.Tail(os.Stdout, -1)
		case "QUIT":
			return
		default:
			fmt.Println(m.styles.err.Render(
				fmt.Sprintf("unrecognised command: %s", strings.Join(cmd, " ")),
			))
		}
	}
}

const programName = "test7800"

func Launch(guiQuit chan bool, ui *ui.UI, args []string) error {
	var bootfile string
	var spec string
	var profile bool
	var bios bool
	var overlay bool

	flgs := flag.NewFlagSet(programName, flag.ExitOnError)
	flgs.StringVar(&spec, "spec", "NTSC", "TV specification of the console: NTSC or PAL")
	flgs.BoolVar(&profile, "profile", false, "create CPU profile for emulator")
	flgs.BoolVar(&bios, "bios", true, "run BIOS routines on reset")
	flgs.BoolVar(&overlay, "overlay", false, "add debugging overlay to display")
	err := flgs.Parse(args)
	if err != nil {
		return err
	}
	args = flgs.Args()

	if len(args) == 1 {
		bootfile = args[0]
	} else if len(args) > 1 {
		return fmt.Errorf("too many arguments to debugger")
	}

	ctx := context{
		console:    "7800",
		spec:       strings.ToUpper(spec),
		useOverlay: overlay,
	}
	ctx.Reset()

	m := &debugger{
		ctx:          ctx,
		guiQuit:      guiQuit,
		state:        ui.State,
		sig:          make(chan os.Signal, 1),
		input:        make(chan input, 1),
		loader:       bootfile,
		styles:       newStyles(),
		breakpoints:  make(map[uint16]bool),
		watches:      make(map[uint16]watch),
		coprocDisasm: &coprocDisasm{},
		coprocDev:    newCoprocDev(),
		bypassBIOS:   !bios,
	}
	m.console = hardware.Create(&m.ctx, ui)

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

	m.loop()

	return nil
}
