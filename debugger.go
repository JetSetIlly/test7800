package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jetsetilly/test7800/disassembly"
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
	console  console
	viewport viewport.Model
	input    textinput.Model
	output   []string
	styles   styles

	// if stopRunning is nil then the console is already stopped
	stopRunning chan bool
}

func (m *debugger) Init() tea.Cmd {
	m.console.initialise()

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

	return nil
}

// step advances the emulation on CPU instruction
func (m *debugger) step() {
	err := m.console.step()
	if err != nil {
		m.output = append(m.output, m.styles.err.Render(
			err.Error(),
		))
	} else {
		res := disassembly.FormatResult(m.console.mc.LastResult)
		m.output = append(m.output, m.styles.instruction.Render(
			strings.TrimSpace(fmt.Sprintf("%s %s %s", res.Address, res.Operator, res.Operand))),
		)
		m.output = append(m.output, m.styles.cpu.Render(
			m.console.mc.String(),
		))
		s := m.console.lastMemoryAccess()
		if len(s) > 0 {
			m.output = append(m.output, m.styles.mem.Render(s))
		}
	}
}

func (m *debugger) run() {
	m.stopRunning = make(chan bool)

	go func() {
		var breakpoint = errors.New("breakpoint")

		hook := func() error {
			pcAddr := m.console.mc.PC.Address()
			if pcAddr == 0xf91a {
				return fmt.Errorf("%w: %04x", breakpoint, pcAddr)
			}
			return nil
		}

		err := m.console.run(m.stopRunning, hook)
		if errors.Is(err, breakpoint) {
			m.output = append(m.output, m.styles.breakpoint.Render(err.Error()))
			m.output = append(m.output, m.styles.cpu.Render(
				m.console.mc.String(),
			))
		}

		close(m.stopRunning)
		m.stopRunning = nil
	}()

	m.output = append(m.output, m.styles.debugger.Render("emulation started"))
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
			if m.stopRunning != nil {
				m.stopRunning <- true
				m.output = append(m.output, m.styles.debugger.Render("emulation stopped"))
				m.output = append(m.output, m.styles.cpu.Render(
					m.console.mc.String(),
				))
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
					err := m.console.reset(true)
					if err != nil {
						m.output = append(m.output, m.styles.err.Render(
							err.Error(),
						))
					} else {
						m.output = append(m.output, m.styles.debugger.Render("console reset"))
					}
				case "CPU":
					m.output = append(m.output, m.styles.cpu.Render(
						m.console.mc.String(),
					))
				case "MARIA":
					m.output = append(m.output, m.styles.mem.Render(
						m.console.mem.maria.Status(),
					))
				case "INPTCTRL":
					m.output = append(m.output, m.styles.mem.Render(
						m.console.mem.inptctrl.Status(),
					))
				case "RAM7800":
					// this dumps the entire contents of RAM to the terminal,
					// which isn't ideal
					for i := 0; i <= len(m.console.mem.ram7800.data)/16; i++ {
						j := i * 15
						m.output = append(m.output, m.styles.mem.Render(
							fmt.Sprintf("% 02x", m.console.mem.ram7800.data[j:j+15]),
						))
					}
				case "PEEK":
					if len(p) < 2 {
						m.output = append(m.output, m.styles.err.Render(
							"PEEK requires an address",
						))
					} else {
						if strings.HasPrefix(p[1], "$") {
							p[1] = fmt.Sprintf("0x%s", p[1][1:])
						}
						addr, err := strconv.ParseUint(p[1], 0, 16)
						if err != nil {
							m.output = append(m.output, m.styles.err.Render(
								fmt.Sprintf("PEEK address is not valid: %s", p[1]),
							))
						} else {
							idx, ar := m.console.mem.mapAddress(uint16(addr))
							if ar == nil {
								m.output = append(m.output, m.styles.err.Render(
									fmt.Sprintf("PEEK address is not in an area: %s", p[1]),
								))
							} else {
								data, err := ar.Read(idx)
								if err != nil {
									m.output = append(m.output, m.styles.err.Render(
										fmt.Sprintf("PEEK address is not readable: %s", p[1]),
									))
								} else {
									m.output = append(m.output, m.styles.mem.Render(
										fmt.Sprintf("$%04x = %02x", addr, data),
									))
								}
							}
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
	return fmt.Sprintf("%s\n%s",
		m.viewport.View(),
		m.input.View(),
	)
}

func startDebugger(endDebugger chan bool) error {
	m := &debugger{}
	p := tea.NewProgram(m)

	go func() {
		<-endDebugger
		p.Quit()
	}()

	return p.Start()
}
