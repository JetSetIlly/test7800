package inptctrl

import (
	"fmt"
)

type INPTCTRL struct {
	value uint8
}

func (ic *INPTCTRL) Label() string {
	return "INPTCTRL"
}

func (ic *INPTCTRL) Status() string {
	return fmt.Sprintf("%s: lock=%v maria=%v bios=%v tia=%v", ic.Label(), ic.Lock(), ic.MARIA(), ic.BIOS(), ic.TIA())
}

func (ic *INPTCTRL) Read(idx uint16) (uint8, error) {
	return ic.value, nil
}

func (ic *INPTCTRL) Write(idx uint16, data uint8) error {
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

func (ic INPTCTRL) BIOS() bool {
	return ic.value&0x04 != 0x04
}

func (ic INPTCTRL) TIA() bool {
	return ic.value&0x08 == 0x08
}
