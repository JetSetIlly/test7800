package bios

import (
	_ "embed"
	"fmt"
)

const OriginBIOS = 0x8000

//go:embed "7800 BIOS (U).rom"
var biosrom []byte

// BIOS files are shorter than the 32k suggested by the origin address in the
// memory map. the adjustment value therefore makes sure Read() and Write()
// index values are correct with respect to the biosrom array
var adjustment uint16

func init() {
	adjustment = uint16((0x10000 - OriginBIOS) - len(biosrom))
}

type BIOS struct {
}

func (b *BIOS) Label() string {
	return "BIOS"
}

func (b *BIOS) Read(idx uint16) (uint8, error) {
	idx -= adjustment
	if int(idx) >= len(biosrom) {
		return 0x00, fmt.Errorf("bios address out of range")
	}
	return biosrom[idx], nil
}

func (b *BIOS) Write(_ uint16, data uint8) error {
	return nil
}
