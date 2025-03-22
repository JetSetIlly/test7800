package cartridge

import "math/rand/v2"

const OriginCart = 0x3000

type Cartridge struct {
}

func (cart *Cartridge) Ejected() bool {
	return true
}

func (cart *Cartridge) Label() string {
	return "Cartridge"
}

func (cart *Cartridge) Read(idx uint16) (uint8, error) {
	// random data because cartridge is ejected. this is required so that the
	// CARTTEST code in the BIOS (run from RAM) fails
	return uint8(rand.IntN(255)), nil
}

func (cart *Cartridge) Write(_ uint16, data uint8) error {
	return nil
}
