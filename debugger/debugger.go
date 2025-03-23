package debugger

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jetsetilly/test7800/disassembly"
	"github.com/jetsetilly/test7800/hardware"
	"github.com/jetsetilly/test7800/hardware/cpu/execution"
	"github.com/jetsetilly/test7800/hardware/maria"
)

type styles struct {
	instruction lipgloss.Style
	cpu         lipgloss.Style
	mem         lipgloss.Style
	video       lipgloss.Style
	err         lipgloss.Style
	breakpoint  lipgloss.Style
	debugger    lipgloss.Style
}

type input struct {
	s   string
	err error
}

type debugger struct {
	externalQuit chan bool
	sig          chan os.Signal
	input        chan input

	console     hardware.Console
	breakpoints map[uint16]bool

	// the boot file to load on console reset
	bootfile string

	// script of commands
	script []string

	// printing styles
	styles styles

	// keep a short history of execution to show when RUN stops
	immediateHistory []execution.Result
}

func (m *debugger) reset() {
	// if a bootfile has been specified on the command line, resetting will use
	// it as part of the reset process. ie. the console will be left in the
	// state directed by the bootfile
	if m.bootfile != "" {
		fmt.Println(m.styles.debugger.Render(
			fmt.Sprintf("booting from %s", m.bootfile),
		))

		var err error
		m.script, err = m.bootFromFile(m.bootfile)
		if err == nil {
			return
		}

		// if there is an error from the bootFromFile() we output it and carry
		// on with the reset as though the bootfile wasn't specified
		fmt.Println(m.styles.err.Render(err.Error()))

		// we also forget about the bootfile because we know it doesn't work
		m.bootfile = ""
	}

	err := m.console.Reset(true)
	if err != nil {
		fmt.Println(m.styles.err.Render(err.Error()))
		return
	}
	fmt.Println(m.styles.debugger.Render("console reset"))
	fmt.Println(m.styles.mem.Render(
		m.console.Mem.BIOS.Status(),
	))
	fmt.Println(m.styles.cpu.Render(
		m.console.MC.String(),
	))
}

// step advances the emulation on CPU instruction
func (m *debugger) step() {
	err := m.console.Step()
	if err != nil {
		fmt.Println(m.styles.err.Render(
			err.Error(),
		))
	} else {
		m.last()
		fmt.Println(m.styles.cpu.Render(
			m.console.MC.String(),
		))
		s := m.console.LastAreaStatus()
		if len(s) > 0 {
			fmt.Println(m.styles.mem.Render(s))
		}
	}
}

func (m *debugger) last() {
	res := disassembly.FormatResult(m.console.MC.LastResult)
	fmt.Println(m.styles.instruction.Render(
		strings.TrimSpace(fmt.Sprintf("%s %s %s", res.Address, res.Operator, res.Operand))),
	)
}

// returns true if quit signal has been received
func (m *debugger) run() bool {
	fmt.Println(m.styles.debugger.Render("emulation running"))

	// we measure the number of instructions in the time period of the running emulation
	var instructionCt int
	var startTime time.Time

	// sentinal errors to
	var (
		breakpointErr = errors.New("breakpoint")
		endRunErr     = errors.New("end run")
		quitErr       = errors.New("quit")
	)

	// hook is called after every CPU instruction
	hook := func() error {
		select {
		case <-m.sig:
			return endRunErr
		case <-m.externalQuit:
			return quitErr
		default:
		}

		// output last instruction
		if m.console.MC.LastResult.Final {
			const immediateHistoryLen = 50
			m.immediateHistory = append(m.immediateHistory, m.console.MC.LastResult)
			if len(m.immediateHistory) > immediateHistoryLen {
				m.immediateHistory = m.immediateHistory[1:]
			}
		}

		instructionCt++
		pcAddr := m.console.MC.PC.Address()
		if _, ok := m.breakpoints[pcAddr]; ok {
			return fmt.Errorf("%w: %04x", breakpointErr, pcAddr)
		}
		if m.console.MARIA.Error != nil {
			if errors.Is(m.console.MARIA.Error, maria.WarningErr) {
				// TODO: output warning
			} else {
				return m.console.MARIA.Error
			}
		}
		return nil
	}

	startTime = time.Now()
	err := m.console.Run(hook)

	if errors.Is(err, quitErr) {
		return true
	}

	// output immediate history on end
	if len(m.immediateHistory) > 0 {
		for _, x := range m.immediateHistory {
			res := disassembly.FormatResult(x)
			fmt.Println(m.styles.instruction.Render(
				strings.TrimSpace(fmt.Sprintf("%s %s %s", res.Address, res.Operator, res.Operand))),
			)
		}
	}

	if errors.Is(err, endRunErr) {
		fmt.Println(m.styles.debugger.Render(
			fmt.Sprintf("%d instructions in %.02f seconds", instructionCt, time.Since(startTime).Seconds())),
		)
	} else if errors.Is(err, breakpointErr) {
		fmt.Println(m.styles.breakpoint.Render(err.Error()))
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
		case <-m.externalQuit:
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
		case "R":
			// shortcut for "RUN"
			m.run()
		case "RUN":
			if m.run() {
				return
			}
		case "STEP":
			m.step()
		case "RESET":
			m.reset()
		case "CPU":
			fmt.Println(m.styles.cpu.Render(
				m.console.MC.String(),
			))
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
			fmt.Println(m.styles.mem.Render(
				m.console.MARIA.DLL.Status(),
			))
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
					fmt.Sprintf("PEEK %s", err.Error()),
				))
				break // switch
			}

			data, err := ma.area.Read(ma.idx)
			if err != nil {
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("PEEK address is not readable: %s", cmd[1]),
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

			// the NEXT argument to BREAK is useful for setting a
			// breakpoint on the instruction on a failed branch
			// instruction, which is a common action when stepping
			// through a program
			//
			// a STEP OVER command would be just as good but we don't
			// have that at the moment
			if strings.ToUpper(cmd[1]) == "NEXT" {
				address := m.console.MC.LastResult.Address
				address += uint16(m.console.MC.LastResult.ByteCount)
				cmd[1] = fmt.Sprintf("%#04x", address)
			}

			ma, err := m.parseAddress(cmd[1])
			if err != nil {
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("BREAK %s", err.Error()),
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
		case "LIST":
			if len(m.breakpoints) == 0 {
				fmt.Println(m.styles.debugger.Render("no breakpoints added"))
			} else {
				for a := range m.breakpoints {
					fmt.Println(m.styles.debugger.Render(fmt.Sprintf("%#04x", a)))
				}
			}
		case "DROP":
			if len(cmd) < 2 {
				fmt.Println(m.styles.err.Render(
					"DROP requires an address",
				))
				break // switch
			}

			ma, err := m.parseAddress(cmd[1])
			if err != nil {
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("DROP %s", err.Error()),
				))
				break // switch
			}

			if _, ok := m.breakpoints[ma.address]; !ok {
				fmt.Println(m.styles.debugger.Render(
					fmt.Sprintf("breakpoint on $%04x does not exist", ma.address),
				))
				break // switch
			}

			delete(m.breakpoints, ma.address)
			fmt.Println(m.styles.debugger.Render(
				fmt.Sprintf("dropped breakpoint for $%04x", ma.address),
			))
		case "QUIT":
			return
		default:
			fmt.Println(m.styles.err.Render(
				fmt.Sprintf("unrecognised command: %s", strings.Join(cmd, " ")),
			))
		}
	}
}

func Launch(externalQuit chan bool, rendering chan *image.RGBA, args []string) error {
	var bootfile string

	if len(args) == 1 {
		bootfile = args[0]
	} else if len(args) > 1 {
		return fmt.Errorf("too many arguments to debugger")
	}

	m := &debugger{
		externalQuit: externalQuit,
		sig:          make(chan os.Signal, 1),
		input:        make(chan input, 1),
		console:      hardware.Create(rendering),
		bootfile:     bootfile,
		styles: styles{
			instruction: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(3)),
			cpu:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(4)),
			mem:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(5)),
			video:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(6)),
			err:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(7)).Background(lipgloss.ANSIColor(1)),
			breakpoint:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(7)).Background(lipgloss.ANSIColor(4)),
			debugger:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(7)).Background(lipgloss.ANSIColor(2)),
		},
		breakpoints: make(map[uint16]bool),
	}

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
	m.loop()

	return nil
}
