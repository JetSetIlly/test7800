package debugger

import "github.com/charmbracelet/lipgloss"

type styles struct {
	instruction lipgloss.Style
	cpu         lipgloss.Style
	mem         lipgloss.Style
	video       lipgloss.Style
	err         lipgloss.Style
	breakpoint  lipgloss.Style
	debugger    lipgloss.Style
	coprocAsm   lipgloss.Style
	coprocCPU   lipgloss.Style
	coprocErr   lipgloss.Style
}

// ANSI Color reference
// 0	Black
// 1	Red
// 2	Green
// 3	Yellow
// 4	Blue
// 5	Magenta
// 6	Cyan
// 7	White
// 8	Bright Black (Gray)
// 9	Bright Red
// 10	Bright Green
// 11	Bright Yellow
// 12	Bright Blue
// 13	Bright Magenta
// 14	Bright Cyan
// 15	Bright White

func newStyles() styles {
	return styles{
		instruction: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(3)),
		cpu:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(4)),
		mem:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(5)),
		video:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(6)),
		err:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(7)).Background(lipgloss.ANSIColor(1)),
		breakpoint:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(7)).Background(lipgloss.ANSIColor(4)),
		debugger:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(7)).Background(lipgloss.ANSIColor(2)),
		coprocAsm:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(0)).Background(lipgloss.ANSIColor(3)),
		coprocCPU:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(7)).Background(lipgloss.ANSIColor(5)),
		coprocErr:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(1)).Background(lipgloss.ANSIColor(7)),
	}
}
