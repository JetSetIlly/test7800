package bios

import _ "embed"

//go:embed "7800 BIOS (U).rom"
var biosrom []byte

type BIOS struct {
}

func (b *BIOS) Label() string {
	return "BIOS"
}

func (b *BIOS) Read(idx uint16) (uint8, error) {
	return biosrom[idx], nil
}

func (b *BIOS) Write(_ uint16, data uint8) error {
	return nil
}
