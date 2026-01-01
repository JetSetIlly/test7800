package debugger

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jetsetilly/dialog"
	"github.com/jetsetilly/test7800/disassembly"
	"github.com/jetsetilly/test7800/hardware/memory"
	"github.com/jetsetilly/test7800/hardware/memory/external"
	"github.com/jetsetilly/test7800/logger"
)

// returns true if debugger is to quit
func (m *debugger) commands(cmd []string) bool {
	if len(cmd) == 0 {
		return false
	}

	switch strings.ToUpper(cmd[0]) {
	case "INSERT":
		if len(cmd) < 2 {
			fmt.Println(m.styles.err.Render(
				"INSERT requires a filename",
			))
			break // switch
		}

		var err error
		m.loader, err = external.Fingerprint(cmd[1], "AUTO")
		if err != nil {
			dialog.Message("Problem with selected file\n\n%v", err).Error()
		} else {
			m.reset()
		}

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
		return m.run()

	case "ST", "STEP":
		if len(cmd) > 1 {
			if !m.parseStepRule(cmd[1:]) {
				break // switch
			}
		} else {
			// step one instruction by default
			m.stepRule = func() bool {
				return true
			}
		}
		return m.run()

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
			m.console.Mem.BIOS.String(),
		))

	case "MARIA":
		fmt.Println(m.styles.mem.Render(
			m.console.MARIA.String(),
		))

	case "DL":
		if len(m.console.MARIA.RecentDL) == 0 {
			fmt.Println(m.styles.mem.Render("no DL activity this scanline"))
		}
		for _, dl := range m.console.MARIA.RecentDL {
			fmt.Println("")
			fmt.Println(m.styles.mem.Render(
				dl.String(),
			))
		}

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
					fmt.Sprintf("unrecognised argument for DLL command: %s", cmd[1]),
				))
			}
		} else {
			fmt.Println(m.styles.mem.Render(
				m.console.MARIA.DLL.String(),
			))
		}

	case "VIDEO":
		fmt.Println(m.styles.video.Render(
			m.console.MARIA.Coords.String(),
		))

	case "INPTCTRL":
		fmt.Println(m.styles.mem.Render(
			m.console.Mem.INPTCTRL.String(),
		))

	case "RAM7800":
		fmt.Println(m.styles.mem.Render(
			m.console.Mem.RAM7800.String(),
		))

	case "RAMRIOT":
		fmt.Println(m.styles.mem.Render(
			m.console.Mem.RAMRIOT.String(),
		))

	case "TIA":
		fmt.Println(m.styles.mem.Render(
			m.console.TIA.String(),
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
			fmt.Sprintf("$%04x = $%02x (%s)", ma.address, data, ma.area.Label()),
		))

	case "POKE":
		if len(cmd) < 3 {
			fmt.Println(m.styles.err.Render(
				"POKE requires an address and a value",
			))
			break // switch
		}

		ma, err := m.parseAddress(cmd[1])
		if err != nil {
			fmt.Println(m.styles.err.Render(
				fmt.Sprintf("poke: %s", err.Error()),
			))
			break // switch
		}

		v, err := strconv.ParseUint(cmd[2], 0, 16)
		if err != nil {
			fmt.Println(m.styles.err.Render(
				fmt.Sprintf("poke: %s", err.Error()),
			))
			break // switch
		}

		err = memory.Write(ma.area, ma.idx, uint8(v))
		if err != nil {
			fmt.Println(m.styles.err.Render(
				fmt.Sprintf("poke address is not writeable: %s", cmd[1]),
			))
			break // switch
		}

		data, err := memory.Read(ma.area, ma.idx)
		if err != nil {
			fmt.Println(m.styles.err.Render(
				fmt.Sprintf("poke address is not readable: %s", cmd[1]),
			))
			break // switch
		}

		fmt.Println(m.styles.mem.Render(
			fmt.Sprintf("$%04x = $%02x (%s)", ma.address, data, ma.area.Label()),
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

		} else if arg == "CONTEXT" {
			m.breakspointCtx = !m.breakspointCtx
			if m.breakspointCtx {
				fmt.Println(m.styles.debugger.Render("context breakpoints enabled"))
			} else {
				fmt.Println(m.styles.debugger.Render("context breakpoints disabled"))
			}
			break // switch
		}

		for i := 1; i < len(cmd); i++ {
			ma, err := m.parseAddress(cmd[i])
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
		}

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

		write := arg == "WRITE"

		// start index for for loop depends on whether the WRITE flag as used
		i := 1
		if write {
			i += 1
		}

		for i := i; i < len(cmd); i++ {
			ma, err := m.parseAddress(cmd[i])
			if err != nil {
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("watch: %s", err.Error()),
				))
				break // switch
			}

			if _, ok := m.watches[ma.address]; ok {
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("watch for %04x already present", ma),
				))
				break // switch
			}

			d, err := memory.Read(ma.area, ma.idx)
			if err != nil {
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("watch address is not readable: %04x", ma),
				))
				break // switch
			}

			m.watches[ma.address] = watch{
				ma:    ma,
				data:  d,
				write: write,
			}
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
		coproc := m.console.Mem.External.GetCoProcBus()
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
		switch len(cmd) {
		case 1:
			logger.Tail(os.Stdout, -1)
		case 2:
			c := strings.ToUpper(cmd[1])
			switch c {
			case "ECHO":
				logger.SetEcho(os.Stdout, false)
			case "NOECHO":
				logger.SetEcho(nil, false)
			default:
				fmt.Println(m.styles.err.Render(
					fmt.Sprintf("unrecognised argument for LOG command: %s", c),
				))
			}
		default:
			fmt.Println(m.styles.err.Render(
				"too many arguments to LOG command",
			))
		}

	case "QUIT":
		return true

	default:
		fmt.Println(m.styles.err.Render(
			fmt.Sprintf("unrecognised command: %s", strings.Join(cmd, " ")),
		))
	}

	return false
}
