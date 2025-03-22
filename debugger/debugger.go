package debugger

import (
	"errors"
	"fmt"
	"image"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	echo        lipgloss.Style
}

type debugger struct {
	console  hardware.Console
	viewport viewport.Model
	input    textinput.Model
	output   []string
	styles   styles

	breakpoints map[uint16]bool

	stopRun chan bool
	running atomic.Bool

	// the boot file to load on console reset
	bootfile string

	// script of commands
	script []string

	// keep a short history of execution
	immediateHistory []execution.Result
}

func (m *debugger) Init() tea.Cmd {
	m.input = textinput.New()
	m.input.Placeholder = ""
	m.input.Focus()
	m.input.CharLimit = 256
	m.input.Width = 50

	m.styles.instruction = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(3))
	m.styles.cpu = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(4))
	m.styles.mem = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(5))
	m.styles.video = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(6))
	m.styles.err = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(7)).Background(lipgloss.ANSIColor(1))
	m.styles.breakpoint = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(7)).Background(lipgloss.ANSIColor(4))
	m.styles.debugger = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(7)).Background(lipgloss.ANSIColor(2))
	m.styles.echo = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(0)).Background(lipgloss.ANSIColor(3))

	m.breakpoints = make(map[uint16]bool)
	m.reset()

	return nil
}

func (m *debugger) reset() {
	// if a bootfile has been specified on the command line, resetting will use
	// it as part of the reset process. ie. the console will be left in the
	// state directed by the bootfile
	if m.bootfile != "" {
		m.output = append(m.output, m.styles.debugger.Render(
			fmt.Sprintf("booting from %s", m.bootfile),
		))

		var err error
		m.script, err = m.bootFromFile(m.bootfile)
		if err == nil {
			return
		}

		// if there is an error from the bootFromFile() we output it and carry
		// on with the reset as though the bootfile wasn't specified
		m.output = append(m.output, m.styles.err.Render(err.Error()))

		// we also forget about the bootfile because we know it doesn't work
		m.bootfile = ""
	}

	err := m.console.Reset(true)
	if err != nil {
		m.output = append(m.output, m.styles.err.Render(err.Error()))
		return
	}
	m.output = append(m.output, m.styles.debugger.Render("console reset"))
	m.output = append(m.output, m.styles.mem.Render(
		m.console.Mem.BIOS.Status(),
	))
	m.output = append(m.output, m.styles.cpu.Render(
		m.console.MC.String(),
	))
}

// step advances the emulation on CPU instruction
func (m *debugger) step() {
	err := m.console.Step()
	if err != nil {
		m.output = append(m.output, m.styles.err.Render(
			err.Error(),
		))
	} else {
		m.last()
		m.output = append(m.output, m.styles.cpu.Render(
			m.console.MC.String(),
		))
		s := m.console.LastAreaStatus()
		if len(s) > 0 {
			m.output = append(m.output, m.styles.mem.Render(s))
		}
	}
}

func (m *debugger) last() {
	res := disassembly.FormatResult(m.console.MC.LastResult)
	m.output = append(m.output, m.styles.instruction.Render(
		strings.TrimSpace(fmt.Sprintf("%s %s %s", res.Address, res.Operator, res.Operand))),
	)
}

const shortHistoryLen = 50

func (m *debugger) run() {
	// the message output at the beginning of the run and that is removed at the
	// end of the run if nothing else has been added to the output. the message
	// is being declared like this and stored in its own variable because we
	// need to check if it is the last message in the output before removing it
	// - the Render() function adds the ANSI codes so it's not really just a
	// plain string
	var emulationRunningMsg = m.styles.debugger.Render("emulation running...")
	m.output = append(m.output, emulationRunningMsg)

	// WARNING: this go function causes race errors but we'll keep it for now
	go func() {
		m.running.Store(true)
		defer m.running.Store(false)

		// we measure the number of instructions in the time period of the
		// running emulation
		var instructionCt int
		var startTime time.Time

		// sentinal error to indicate a breakpoint has been encountered
		var breakpointErr = errors.New("breakpoint")

		// hook is called after every CPU instruction
		hook := func() error {
			// keeping a short history of execution
			if m.console.MC.LastResult.Final {
				m.immediateHistory = append(m.immediateHistory, m.console.MC.LastResult)
				if len(m.immediateHistory) > shortHistoryLen {
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
		err := m.console.Run(m.stopRun, hook)

		// replace the last entry in the output (which should be "emulation
		// running...") with an instruction/time summary
		if m.output[len(m.output)-1] == emulationRunningMsg {
			m.output[len(m.output)-1] = m.styles.debugger.Render(
				fmt.Sprintf("%d instructions in %.02f seconds", instructionCt, time.Since(startTime).Seconds()),
			)
		}

		if len(m.immediateHistory) > 0 {
			for _, x := range m.immediateHistory {
				res := disassembly.FormatResult(x)
				m.output = append(m.output, m.styles.instruction.Render(
					strings.TrimSpace(fmt.Sprintf("%s %s %s", res.Address, res.Operator, res.Operand))),
				)
			}
		}

		if err != nil {
			if errors.Is(err, breakpointErr) {
				m.output = append(m.output, m.styles.breakpoint.Render(err.Error()))
			} else {
				m.output = append(m.output, m.styles.err.Render(err.Error()))
			}
		}

		// it's useful to see the state of the CPU and the MARIA coords at the end of the run
		m.output = append(m.output, m.styles.cpu.Render(
			m.console.MC.String(),
		))
		m.output = append(m.output, m.styles.video.Render(
			m.console.MARIA.Coords.String(),
		))

		// consume last memory access information
		_ = m.console.LastAreaStatus()
	}()
}

func (m *debugger) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 1

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			// stop any running emulation OR quit the application
			if m.running.Load() {
				m.stopRun <- true
			} else {
				return m, tea.Quit
			}
		case "enter":
			s := m.input.Value()
			s = strings.TrimSpace(s)
			p := strings.Fields(s)
			if len(p) == 0 {
				m.step()
			} else {
				cmd := strings.ToUpper(p[0])
				m.output = append(m.output, m.styles.echo.Render(
					strings.TrimSpace(
						fmt.Sprintf("%s %s", cmd, strings.Join(p[1:], " ")),
					),
				))
				switch cmd {
				case "BOOT":
					if len(p) < 5 {
						m.output = append(m.output, m.styles.err.Render(
							"BOOT requires a ROM file, an origin address, an entry address and the INPTCTRL value",
						))
						break // switch
					}

					err := m.bootParse(p[1:])
					if err != nil {
						m.output = append(m.output, m.styles.err.Render(err.Error()))
						break // switch
					}
				case "R":
					// shortcut for "RUN"
					m.run()
				case "RUN":
					m.run()
				case "STEP":
					m.step()
				case "RESET":
					m.reset()
				case "CPU":
					m.output = append(m.output, m.styles.cpu.Render(
						m.console.MC.String(),
					))
				case "BIOS":
					m.output = append(m.output, m.styles.mem.Render(
						m.console.Mem.BIOS.Status(),
					))
				case "MARIA":
					m.output = append(m.output, m.styles.mem.Render(
						m.console.MARIA.Status(),
					))
				case "DL":
					m.output = append(m.output, m.styles.mem.Render(
						m.console.MARIA.DL.Status(),
					))
				case "DLL":
					m.output = append(m.output, m.styles.mem.Render(
						m.console.MARIA.DLL.Status(),
					))
				case "VIDEO":
					m.output = append(m.output, m.styles.video.Render(
						m.console.MARIA.Coords.String(),
					))
				case "INPTCTRL":
					m.output = append(m.output, m.styles.mem.Render(
						m.console.Mem.INPTCTRL.Status(),
					))
				case "RAM7800":
					m.output = append(m.output, m.styles.mem.Render(
						m.console.Mem.RAM7800.String(),
					))
				case "RAMRIOT":
					m.output = append(m.output, m.styles.mem.Render(
						m.console.Mem.RAMRIOT.String(),
					))
				case "PEEK":
					if len(p) < 2 {
						m.output = append(m.output, m.styles.err.Render(
							"PEEK requires an address",
						))
						break // switch
					}

					ma, err := m.parseAddress(p[1])
					if err != nil {
						m.output = append(m.output, m.styles.err.Render(
							fmt.Sprintf("PEEK %s", err.Error()),
						))
						break // switch
					}

					data, err := ma.area.Read(ma.idx)
					if err != nil {
						m.output = append(m.output, m.styles.err.Render(
							fmt.Sprintf("PEEK address is not readable: %s", p[1]),
						))
						break // switch
					}

					m.output = append(m.output, m.styles.mem.Render(
						fmt.Sprintf("$%04x = %02x (%s)", ma.address, data, ma.area.Label()),
					))
				case "BREAK":
					if len(p) < 2 {
						m.output = append(m.output, m.styles.err.Render(
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
					if strings.ToUpper(p[1]) == "NEXT" {
						address := m.console.MC.LastResult.Address
						address += uint16(m.console.MC.LastResult.ByteCount)
						p[1] = fmt.Sprintf("%#04x", address)
					}

					ma, err := m.parseAddress(p[1])
					if err != nil {
						m.output = append(m.output, m.styles.err.Render(
							fmt.Sprintf("BREAK %s", err.Error()),
						))
						break // switch
					}

					if _, ok := m.breakpoints[ma.address]; ok {
						m.output = append(m.output, m.styles.debugger.Render(
							fmt.Sprintf("breakpoint on $%04x already present", ma.address),
						))
						break // switch
					}

					m.breakpoints[ma.address] = true
					m.output = append(m.output, m.styles.debugger.Render(
						fmt.Sprintf("added breakpoint for $%04x", ma.address),
					))
				case "LIST":
					if len(m.breakpoints) == 0 {
						m.output = append(m.output, m.styles.debugger.Render("no breakpoints added"))
					} else {
						for a := range m.breakpoints {
							m.output = append(m.output, m.styles.debugger.Render(fmt.Sprintf("%#04x", a)))
						}
					}
				case "DROP":
					if len(p) < 2 {
						m.output = append(m.output, m.styles.err.Render(
							"DROP requires an address",
						))
						break // switch
					}

					ma, err := m.parseAddress(p[1])
					if err != nil {
						m.output = append(m.output, m.styles.err.Render(
							fmt.Sprintf("DROP %s", err.Error()),
						))
						break // switch
					}

					if _, ok := m.breakpoints[ma.address]; !ok {
						m.output = append(m.output, m.styles.debugger.Render(
							fmt.Sprintf("breakpoint on $%04x does not exist", ma.address),
						))
						break // switch
					}

					delete(m.breakpoints, ma.address)
					m.output = append(m.output, m.styles.debugger.Render(
						fmt.Sprintf("dropped breakpoint for $%04x", ma.address),
					))
				case "QUIT":
					return m, tea.Quit
				default:
					m.output = append(m.output, m.styles.err.Render(
						fmt.Sprintf("unrecognised command: %s", s),
					))
				}
			}

			m.input.SetValue("")
		}
	}

	// always update viewport and scroll to bottom. this isn't optimal and means
	// we can't scroll the viewport up but this is the best I can do for now
	m.viewport.SetContent(strings.Join(m.output, "\n"))
	m.viewport.GotoBottom()

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	// execute script
	if len(m.script) > 0 {
		m.input.SetValue(m.script[0])
		cmds = append(cmds,
			func() tea.Msg { return tea.KeyMsg{Type: tea.KeyEnter} })
		m.script = m.script[1:]
	}

	return m, tea.Batch(cmds...)
}

func (m *debugger) View() string {
	if m.running.Load() {
		return fmt.Sprintf("%s\n%s",
			m.viewport.View(),
			m.console.MARIA.Coords.ShortString(),
		)
	}
	m.input.Prompt = fmt.Sprintf("%s > ", m.console.MARIA.Coords.ShortString())
	return fmt.Sprintf("%s\n%s",
		m.viewport.View(),
		m.input.View(),
	)
}

func Launch(endDebugger chan bool, rendering chan *image.RGBA, args []string) error {
	var bootfile string

	if len(args) == 1 {
		bootfile = args[0]
	} else if len(args) > 1 {
		return fmt.Errorf("too many arguments to debugger")
	}

	m := &debugger{
		console:  hardware.Create(rendering),
		bootfile: bootfile,
		stopRun:  make(chan bool),
	}
	p := tea.NewProgram(m)

	go func() {
		<-endDebugger
		p.Quit()
	}()

	return p.Start()
}
