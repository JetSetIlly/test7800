package bios

import (
	"crypto/md5"
	_ "embed"
	"fmt"
)

// the addressing origin for the BIOS. the bios rom data will not be placed here
// unless the BIOS is 32k in length
const OriginBIOS = 0x8000

// the largest BIOS size possible
const maxBIOSsize = 0x10000 - OriginBIOS

//go:embed "7800 BIOS (E).rom"
var biosrom []byte

// "7800 BIOS (U).rom" is the NTSC ROM
// "7800 BIOS (E).rom" is the PAL ROM

// list of known BIOS checksums (md5) and the name
var knownBIOS [][2]string = [][2]string{
	{"PAL", "0x397bb566584be7b9764e7a68974c4263"},
	{"NTSC", "0x0763f1ffb006ddbe32e52d497ee848ae"},
}

// which BIOS has been detected and loaded
var spec string

// the actual origin point for the specified BIOS rom
var origin uint16

// BIOS files tend to be shorter than the 32k suggested by the origin address in
// the memory map. the adjustment value makes sure Read() and Write() index
// values are correct in relation to the actual bios data
var adjustment uint16

func init() {
	sz := len(biosrom)
	if sz > maxBIOSsize {
		panic(fmt.Sprintf("specified BIOS rom is too large: %d but max is %d", sz, maxBIOSsize))
	}

	adjustment = uint16(maxBIOSsize - sz)
	origin = uint16(0x10000 - sz)

	// double check size of BIOS
	if origin < OriginBIOS {
		panic(
			fmt.Sprintf("specified BIOS rom is too large: placed at origin %#04x. lowest possible %#04x", origin, OriginBIOS),
		)
	}

	// default spec is unknown
	spec = "unknown"

	// find the BIOS spec
	h := fmt.Sprintf("%#16x", md5.Sum(biosrom))
	for _, k := range knownBIOS {
		if k[1] == h {
			spec = k[0]
			break // for loop
		}
	}
}

type BIOS struct {
	// bios type is intionally empty. it would be an improvement for the BIOS
	// type to contain the origin, adjustment, etc. fields that are currently
	// package wide
}

func (b *BIOS) Spec() string {
	return spec
}

func (b *BIOS) Label() string {
	return "BIOS"
}

func (b *BIOS) Status() string {
	return fmt.Sprintf("%dk %s BIOS at %#04x", len(biosrom)/1024, spec, origin)
}

func (b *BIOS) Read(idx uint16) (uint8, error) {
	// check that the mapping process hasn't given us an index that is an
	// impossible address for the BIOS. this shouldn't ever happen
	if int(idx) >= maxBIOSsize {
		return 0x00, fmt.Errorf("bios address out of range")
	}

	idx -= adjustment

	// check that index is inside the acutal size of the loaded bios. if it is
	// not then return 0x00 without error. this is correct because the index is
	// still pointing to a BIOS address, we just don't have any data for it.
	// it's unclear what value the real hardware returns but whatever it is,
	// it's not an error
	if int(idx) >= len(biosrom) {
		return 0x00, nil
	}

	return biosrom[idx], nil
}

func (b *BIOS) Write(_ uint16, data uint8) error {
	return nil
}
