package debugger

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jetsetilly/test7800/disassembly"
	"github.com/jetsetilly/test7800/hardware"
)

type styles struct {
	instruction lipgloss.Style
	cpu         lipgloss.Style
	mem         lipgloss.Style
	err         lipgloss.Style
	breakpoint  lipgloss.Style
	debugger    lipgloss.Style
}

type debugger struct {
	console  hardware.Console
	viewport viewport.Model
	input    textinput.Model
	output   []string
	styles   styles

	breakpoints map[uint16]bool

	// if stopRun is nil then the emulated console is already stopped
	stopRun chan bool
}

func (m *debugger) Init() tea.Cmd {
	m.console = hardware.Create()

	m.input = textinput.New()
	m.input.Placeholder = ""
	m.input.Focus()
	m.input.CharLimit = 256
	m.input.Width = 50

	m.styles.instruction = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(3))
	m.styles.cpu = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(4))
	m.styles.mem = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(5))
	m.styles.err = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(7)).Background(lipgloss.ANSIColor(1))
	m.styles.breakpoint = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(7)).Background(lipgloss.ANSIColor(4))
	m.styles.debugger = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(7)).Background(lipgloss.ANSIColor(2))

	m.breakpoints = make(map[uint16]bool)

	return nil
}

// step advances the emulation on CPU instruction
func (m *debugger) step() {
	err := m.console.Step()
	if err != nil {
		m.output = append(m.output, m.styles.err.Render(
			err.Error(),
		))
	} else {
		res := disassembly.FormatResult(m.console.MC.LastResult)
		m.output = append(m.output, m.styles.instruction.Render(
			strings.TrimSpace(fmt.Sprintf("%s %s %s", res.Address, res.Operator, res.Operand))),
		)
		m.output = append(m.output, m.styles.cpu.Render(
			m.console.MC.String(),
		))
		s := m.console.LastMemoryAccess()
		if len(s) > 0 {
			m.output = append(m.output, m.styles.mem.Render(s))
		}
	}
}

func (m *debugger) run() {
	m.stopRun = make(chan bool)
	m.output = append(m.output, m.styles.debugger.Render("emulation running..."))

	go func() {
		// we measure the number of instructions in the time period of the
		// running emulation
		var instructions int
		var startTime time.Time

		// sentinal error to indicate a breakpoint has been encountered
		var breakpoint = errors.New("breakpoint")

		// hook is called after every CPU instruction
		hook := func() error {
			instructions++
			pcAddr := m.console.MC.PC.Address()
			if _, ok := m.breakpoints[pcAddr]; ok {
				return fmt.Errorf("%w: %04x", breakpoint, pcAddr)
			}
			return nil
		}

		startTime = time.Now()
		err := m.console.Run(m.stopRun, hook)

		// replace the last entry in the output (which should be "emulation
		// running...") with an instruction/time summary
		m.output[len(m.output)-1] = m.styles.debugger.Render(
			fmt.Sprintf("%d instructions in %.02f seconds", instructions, time.Since(startTime).Seconds()),
		)

		if errors.Is(err, breakpoint) {
			m.output = append(m.output, m.styles.breakpoint.Render(err.Error()))
		} else {
			m.output = append(m.output, m.styles.err.Render(err.Error()))
		}

		// it's useful to see the state of the CPU at the end of the run
		m.output = append(m.output, m.styles.cpu.Render(
			m.console.MC.String(),
		))

		close(m.stopRun)
		m.stopRun = nil

		// consume last memory access information
		_ = m.console.LastMemoryAccess()
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
			if m.stopRun != nil {
				m.stopRun <- true
			} else {
				return m, tea.Quit
			}
		case "enter":
			s := m.input.Value()
			s = strings.TrimSpace(s)
			s = strings.ToUpper(s)

			p := strings.Fields(s)
			if len(p) == 0 {
				m.step()
			} else {
				switch p[0] {
				case "RUN":
					m.run()
				case "STEP":
					m.step()
				case "RESET":
					err := m.console.Reset(true)
					if err != nil {
						m.output = append(m.output, m.styles.err.Render(
							err.Error(),
						))
					} else {
						m.output = append(m.output, m.styles.debugger.Render("console reset"))
					}
				case "CPU":
					m.output = append(m.output, m.styles.cpu.Render(
						m.console.MC.String(),
					))
				case "MARIA":
					m.output = append(m.output, m.styles.mem.Render(
						m.console.Mem.MARIA.Status(),
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
					} else {
						ma, err := m.parseAddress(p[1])
						if err != nil {
							m.output = append(m.output, m.styles.err.Render(
								fmt.Sprintf("PEEK %s", err.Error()),
							))
						}

						data, err := ma.area.Read(ma.idx)
						if err != nil {
							m.output = append(m.output, m.styles.err.Render(
								fmt.Sprintf("PEEK address is not readable: %s", p[1]),
							))
						} else {
							m.output = append(m.output, m.styles.mem.Render(
								fmt.Sprintf("$%04x = %02x", ma.address, data),
							))
						}
					}
				case "BREAK":
					if len(p) < 2 {
						m.output = append(m.output, m.styles.err.Render(
							"BREAK requires an address",
						))
					} else {
						ma, err := m.parseAddress(p[1])
						if err != nil {
							m.output = append(m.output, m.styles.err.Render(
								fmt.Sprintf("BREAK %s", err.Error()),
							))
						}

						if _, ok := m.breakpoints[ma.address]; ok {
							m.output = append(m.output, m.styles.debugger.Render(
								fmt.Sprintf("breakpoint on $%04x already present", ma.address),
							))
						} else {
							m.breakpoints[ma.address] = true
							m.output = append(m.output, m.styles.debugger.Render(
								fmt.Sprintf("added breakpoint for $%04x", ma.address),
							))
						}
					}
				case "DROP":
					if len(p) < 2 {
						m.output = append(m.output, m.styles.err.Render(
							"DROP requires an address",
						))
					} else {
						ma, err := m.parseAddress(p[1])
						if err != nil {
							m.output = append(m.output, m.styles.err.Render(
								fmt.Sprintf("DROP %s", err.Error()),
							))
						}

						if _, ok := m.breakpoints[ma.address]; ok {
							delete(m.breakpoints, ma.address)
							m.output = append(m.output, m.styles.debugger.Render(
								fmt.Sprintf("dropped breakpoint for $%04x", ma.address),
							))
						} else {
							m.output = append(m.output, m.styles.debugger.Render(
								fmt.Sprintf("breakpoint on $%04x does not exist", ma.address),
							))
						}
					}
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

	return m, tea.Batch(cmds...)
}

func (m debugger) View() string {
	// if emulation is running (ie. stopEmulation is not nil) then we don't want
	// to allow user input because that might lead to a data-race. for example,
	// PEEKing an address will likely collide with memory access by the
	// emulation
	if m.stopRun == nil {
		return fmt.Sprintf("%s\n%s",
			m.viewport.View(),
			m.input.View(),
		)
	}
	return m.viewport.View()
}

func Launch(endDebugger chan bool) error {
	m := &debugger{}
	p := tea.NewProgram(m)

	go func() {
		<-endDebugger
		p.Quit()
	}()

	return p.Start()
}
