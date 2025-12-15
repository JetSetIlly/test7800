package debugger

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func (m *debugger) bootFromFile(d []byte) ([]string, error) {
	lns := strings.Split(strings.TrimSpace(string(d)), "\n")
	if len(lns) < 1 {
		return []string{}, fmt.Errorf("there must be at least one line in a bootfile")
	}

	p := strings.Fields(lns[0])
	if len(p) > 4 {
		return []string{}, fmt.Errorf("too many fields in bootfile")
	}

	if len(p) < 4 {
		return []string{}, fmt.Errorf("too few fields in bootfile")
	}

	err := m.bootParse(p)
	if err != nil {
		return []string{}, err
	}

	// use remainder of the file as a boot script. blank lines are not accepted
	// in the script and are filtered out
	var script []string
	for _, l := range lns[1:] {
		l = strings.TrimSpace(l)
		if l != "" {
			script = append(script, l)
		}
	}

	return script, nil
}

func (m *debugger) bootParse(args []string) error {
	origin, err := m.parseAddress(args[1])
	if err != nil {
		return err
	}

	entry, err := m.parseAddress(args[2])
	if err != nil {
		return err
	}

	inptctrl, err := strconv.ParseUint(args[3], 0, 8)
	if err != nil {
		return err
	}

	return m.boot(args[0], origin, entry, uint8(inptctrl))
}

// loads a ROM file at the stated origin and sets the PC accordingly
func (m *debugger) boot(romfile string, origin mappedAddress, entry mappedAddress, inptctrl uint8) error {
	d, err := os.ReadFile(romfile)
	if err != nil {
		return fmt.Errorf("error loading %s", romfile)
	}

	// the console may already have been reset but we'll reset it again to make sure
	err = m.console.Reset(true, nil)
	if err != nil {
		return err
	}

	// copy romfile into memory a the origin address. if the memory is read-only
	// then the console has been reset
	for i, b := range d {
		_, err := origin.area.Access(true, origin.idx+uint16(i), b)
		if err != nil {
			return err
		}
	}

	// first instruction at entry point
	m.console.MC.PC.Load(entry.address)

	// writing to the INPTCTRL twice to make sure the halt line has been enabled
	m.console.Mem.INPTCTRL.Write(0x01, inptctrl)
	m.console.Mem.INPTCTRL.Write(0x01, inptctrl)

	fmt.Println(m.styles.debugger.Render(
		fmt.Sprintf("loaded %s at %#04x", romfile, origin.address),
	))
	fmt.Println(m.styles.cpu.Render(
		m.console.Mem.INPTCTRL.Status(),
	))
	fmt.Println(m.styles.cpu.Render(
		m.console.MC.String(),
	))
	return nil
}
