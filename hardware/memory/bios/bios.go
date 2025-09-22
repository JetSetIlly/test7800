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
var pal []byte

//go:embed "7800 BIOS (U).rom"
var ntsc []byte

// list of known BIOS checksums (md5) and the name
var knownBIOS map[string]string = map[string]string{
	// "7800 BIOS (E).rom" is the PAL ROM
	"PAL": "0x397bb566584be7b9764e7a68974c4263",

	// "7800 BIOS (U).rom" is the NTSC ROM
	"NTSC": "0x0763f1ffb006ddbe32e52d497ee848ae",
}

func init() {
	sz := len(pal)
	if sz > maxBIOSsize {
		panic(fmt.Sprintf("PAL BIOS rom is too large: %d but max is %d", sz, maxBIOSsize))
	}

	sz = len(ntsc)
	if sz > maxBIOSsize {
		panic(fmt.Sprintf("NTSC BIOS rom is too large: %d but max is %d", sz, maxBIOSsize))
	}

	h := fmt.Sprintf("%#16x", md5.Sum(pal))
	if h != knownBIOS["PAL"] {
		panic("unsupported PAL bios")
	}

	h = fmt.Sprintf("%#16x", md5.Sum(ntsc))
	if h != knownBIOS["NTSC"] {
		panic("unsupported NTSC bios")
	}
}

type BIOS struct {
	spec       string
	origin     uint16
	adjustment uint16
	data       []byte
}

func NewPAL() BIOS {
	b := BIOS{
		spec: "PAL",
		data: pal,
	}

	b.origin = uint16(0x10000 - len(pal))
	b.adjustment = uint16(maxBIOSsize - len(pal))

	// double check size of BIOS
	if b.origin < OriginBIOS {
		panic(
			fmt.Sprintf("%s BIOS rom is too large: placed at origin %#04x. lowest possible %#04x", b.spec, b.origin, OriginBIOS),
		)
	}

	return b
}

func NewNTSC() BIOS {
	b := BIOS{
		spec: "NTSC",
		data: ntsc,
	}

	b.origin = uint16(0x10000 - len(ntsc))
	b.adjustment = uint16(maxBIOSsize - len(ntsc))

	// double check size of BIOS
	if b.origin < OriginBIOS {
		panic(
			fmt.Sprintf("%s BIOS rom is too large: placed at origin %#04x. lowest possible %#04x", b.spec, b.origin, OriginBIOS),
		)
	}

	return b
}

// return MD5 sum of BIOS
func (b *BIOS) MD5() string {
	return fmt.Sprintf("%#16x", md5.Sum(b.data))
}

func (b *BIOS) Label() string {
	return "BIOS"
}

func (b *BIOS) Status() string {
	return b.String()
}

func (b *BIOS) String() string {
	return fmt.Sprintf("%dk %s BIOS at %#04x", len(pal)/1024, b.spec, b.origin)
}

func (b *BIOS) Access(write bool, address uint16, data uint8) (uint8, error) {
	if write {
		return data, nil
	}
	if address < b.origin {
		return 0, nil
	}
	idx := address - OriginBIOS - b.adjustment
	return b.data[idx], nil
}
