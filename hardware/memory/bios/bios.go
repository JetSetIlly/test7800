package bios

import (
	"crypto/md5"
	_ "embed"
	"fmt"
)

// the addressing origin for the BIOS. the bios rom data will not be placed here
// unless the BIOS is 32k in length
//
// the reason for this origin is given in the "7800 Software Guide" in the
// section detailing the INPTCTRL register and the EXT bit in particular:
//
// "EXT: 0 = enable BIOS at $8000-$FFFF, 1 = disable BIOS / enable cartridge"
const OriginBIOS = 0x8000

// the largest BIOS size possible
const maxBIOSsize = 0x10000 - OriginBIOS

//go:embed "7800 BIOS (E).rom"
var biosrom []byte

// list of known BIOS checksums (md5) and the name
var knownBIOS map[string]string = map[string]string{
	// "7800 BIOS (E).rom" is the PAL ROM
	"PAL": "0x397bb566584be7b9764e7a68974c4263",

	// "7800 BIOS (U).rom" is the NTSC ROM
	"NTSC": "0x0763f1ffb006ddbe32e52d497ee848ae",
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
	for spc, hsh := range knownBIOS {
		if h == hsh {
			spec = spc
			break // for loop
		}
	}
}

type BIOS struct {
	// bios type is intionally empty. it would be an improvement for the BIOS
	// type to contain the origin, adjustment, etc. and other fields that are
	// currently package wide
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

func (b *BIOS) Access(write bool, address uint16, data uint8) (uint8, error) {
	if write {
		return data, nil
	}
	if address < origin {
		return 0, nil
	}
	idx := address - OriginBIOS - adjustment
	return biosrom[idx], nil
}
