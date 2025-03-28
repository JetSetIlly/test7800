package debugger

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jetsetilly/test7800/hardware/cpu/instructions"
)

func (m *debugger) parseStepRule(cmd []string) bool {
	// rough support for step rule definition

	rule := strings.ToUpper(cmd[0])
	if rule == "FRAME" || rule == "FR" {
		var tgt int
		if len(cmd) > 1 {
			var err error
			tgt, err = strconv.Atoi(cmd[1])
			if err != nil {
				fmt.Println(m.styles.err.Render(err.Error()))
				return false
			}
			if tgt <= m.console.MARIA.Coords.Frame {
				fmt.Println(m.styles.err.Render(fmt.Sprintf("FRAME %d is in the past", tgt)))
				return false
			}
		} else {
			tgt = m.console.MARIA.Coords.Frame + 1
		}
		m.stepRule = func() bool {
			return m.console.MARIA.Coords.Frame == tgt
		}
	} else if rule == "SCANLINE" || rule == "SL" {
		var tgt int
		if len(cmd) > 1 {
			var err error
			tgt, err = strconv.Atoi(cmd[1])
			if err != nil {
				fmt.Println(m.styles.err.Render(err.Error()))
				return false
			}
			if tgt <= m.console.MARIA.Coords.Scanline {
				fmt.Println(m.styles.err.Render(fmt.Sprintf("SCANLINE %d is in the past", tgt)))
				return false
			}
		} else {
			tgt = m.console.MARIA.Coords.Scanline + 1
		}
		m.stepRule = func() bool {
			return m.console.MARIA.Coords.Scanline == tgt
		}
	} else if rule == "INTERRUPT" || rule == "INTR" {
		// steps to the next instruction which is inside an
		// interrupt. if consecutive instructions are in an
		// interrupt then this is effectively the same as step
		m.stepRule = func() bool {
			return m.console.MC.LastResult.InInterrupt
		}
	} else if rule == "DLL" {
		id := m.console.MARIA.DLL.ID()
		m.stepRule = func() bool {
			return id != m.console.MARIA.DLL.ID()
		}
		m.postStep = func() {
			fmt.Println(m.styles.mem.Render(
				m.console.MARIA.DLL.Status(),
			))
		}
	} else {
		// check if rule is in a CPU operator
		var found bool
		for _, d := range instructions.Definitions {
			if rule == strings.ToUpper(d.Operator.String()) {
				found = true
				break
			}
		}
		if !found {
			fmt.Println(m.styles.err.Render(
				fmt.Sprintf("STEP %s is unsupported", rule),
			))
			return false
		}
		m.stepRule = func() bool {
			op := strings.ToUpper(m.console.MC.LastResult.Defn.Operator.String())
			return op == rule
		}
	}
	return true
}
