package inptctrl

import "fmt"

type INPTCTRL struct {
	value uint8
}

func (ic *INPTCTRL) Label() string {
	return "INPTCTRL"
}

func (ic *INPTCTRL) Status() string {
	lock := ic.value&0x01 == 0x01
	maria := ic.value&0x02 == 0x02
	bios := ic.value&0x04 == 0x04
	tia := ic.value&0x08 == 0x08
	return fmt.Sprintf("%s: lock=%v maria=%v bios=%v tia=%v", ic.Label(), lock, maria, bios, tia)
}

func (ic *INPTCTRL) Read(_ uint16) (uint8, error) {
	return ic.value, nil
}

func (ic *INPTCTRL) Write(_ uint16, data uint8) error {
	if ic.value&0x01 == 0x01 {
		return nil
	}
	ic.value = data
	return nil
}
