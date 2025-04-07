package inptctrl

import (
	"fmt"
)

type INPTCTRL struct {
	value uint8

	// "In addition to the functions controlled by the register bits, INPTCTRL
	// also controls the HALT input to the 6502. When the 7800 is first powered
	// up HALT is blocked from getting to the 6502, but after 2 writes to the
	// control register (the data written doesn’t make a difference), the HALT
	// input will be enabled."
	enableHalt int
}

func (ic *INPTCTRL) Reset() {
	ic.value = 0
	ic.enableHalt = 0
}

func (ic *INPTCTRL) Label() string {
	return "INPTCTRL"
}

func (ic *INPTCTRL) Status() string {
	return fmt.Sprintf("%s: lock=%v maria=%v bios=%v tia=%v", ic.Label(), ic.Lock(), ic.MARIA(), ic.BIOS(), ic.TIA())
}

func (ic *INPTCTRL) Access(write bool, idx uint16, data uint8) (uint8, error) {
	if write {
		return data, ic.Write(idx, data)
	}
	return ic.Read(idx)
}

func (ic *INPTCTRL) Read(idx uint16) (uint8, error) {
	return ic.value, nil
}

func (ic *INPTCTRL) Write(idx uint16, data uint8) error {
	if ic.enableHalt < 2 {
		ic.enableHalt++
	}
	if ic.Lock() {
		return nil
	}
	ic.value = data
	return nil
}

func (ic INPTCTRL) Lock() bool {
	return ic.value&0x01 == 0x01
}

func (ic INPTCTRL) MARIA() bool {
	return ic.value&0x02 == 0x02
}

func (ic INPTCTRL) EXT() bool {
	return ic.value&0x04 == 0x04
}

// BIOS function returns the inverted meaning of the EXT bit in the INPTCTRL register
func (ic INPTCTRL) BIOS() bool {
	return !ic.EXT()
}

func (ic INPTCTRL) TIA() bool {
	return ic.value&0x08 == 0x08
}

func (ic *INPTCTRL) HaltEnabled() bool {
	return ic.enableHalt > 1
}
