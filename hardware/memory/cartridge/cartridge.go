package cartridge

import (
	_ "embed"
	"fmt"
)

const OriginCart = 0x3000

// the largest cartridge size possible
const maxCartSize = 0x10000 - OriginCart

//go:embed "balldem2.a78"
var data []byte

// the actual origin point for the loaded cartridge
var origin uint16

// the adjustment value makes sure Read() and Write() index values are correct
// in relation to the actual bios data
var adjustment uint16

type Cartridge struct {
	ctx Context
}

type Context interface {
	Rand8Bit() uint8
}

func Create(ctx Context) *Cartridge {
	return &Cartridge{
		ctx: ctx,
	}
}

func init() {
	sz := len(data)
	if sz > maxCartSize {
		panic(fmt.Sprintf("specified cartridge is too large: %d but max is %d", sz, maxCartSize))
	}

	adjustment = uint16(maxCartSize - sz)
	origin = uint16(0x10000 - sz)

	// double check size of BIOS
	if origin < OriginCart {
		panic(
			fmt.Sprintf("specified cartridge is too large: placed at origin %#04x. lowest possible %#04x", origin, OriginCart),
		)
	}
}

func (cart *Cartridge) Label() string {
	return "Cartridge"
}

// whether to treat the embedded cartridge as ejected or inserted
const ejected = true

func (cart *Cartridge) Read(idx uint16) (uint8, error) {
	if ejected {
		return cart.ctx.Rand8Bit(), nil
	}

	// check that the mapping process hasn't given us an index that is an
	// impossible address for the BIOS. this shouldn't ever happen
	if int(idx) >= maxCartSize {
		return 0x00, fmt.Errorf("cartridge address out of range")
	}

	idx -= adjustment

	// check that index is inside the acutal size of the loaded bios. if it is
	// not then return 0x00 without error. this is correct because the index is
	// still pointing to a BIOS address, we just don't have any data for it.
	// it's unclear what value the real hardware returns but whatever it is,
	// it's not an error
	if int(idx) >= len(data) {
		return 0x00, nil
	}

	return data[idx], nil
}

func (cart *Cartridge) Write(_ uint16, data uint8) error {
	return nil
}
