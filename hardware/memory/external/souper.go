package external

import (
	"fmt"
)

// Developed using verilog files as a reference. The zip files containing the verilog source are at
// the following link:
//
// https://forums.atariage.com/topic/285923-souper-mapper/
//
// Also at the link is a link to a file named 'EQU_SOUPER.asm' which was useful for making clear how
// the ChrA/B bank selection is done (bit 8 of the lower address byte) and that the fixed bank for
// maria is different to the fixed bank for sally
//
// Any comments below that are in quotation marks are from the verilog files

type Souper struct {
	data     []byte
	numBanks int
	bank     int

	// base address for the sally and maria fixed banks
	sallyFixedBase uint32
	mariaFixedBase uint32

	// 32k of RAM
	ram   [0x8000]byte
	vBank uint32
	dBank uint32

	// addressing value set by chrA and chrB registers. the address is completed with the actual
	// address from the access
	chrA uint32
	chrB uint32

	// maria access
	hlt bool

	// mode register controls how ROM/RAM is accessed
	mode uint8
}

func NewSouper(_ Context, d []byte) (*Souper, error) {
	if len(d)%16384 != 0 {
		return nil, fmt.Errorf("illegal size")
	}

	ext := &Souper{
		data:     d[:],
		numBanks: len(d) / 16384,
	}

	// it wasn't clear to me that the fixed bank for maria accesses was different to the fixed bank
	// for sally accesses. the maria fixed bank is used for the text in the game scoreline of
	// rikki-and-vikki
	ext.sallyFixedBase = uint32(ext.numBanks-2) << 14
	ext.mariaFixedBase = uint32(ext.numBanks-1) << 14

	return ext, nil
}

func (ext *Souper) Label() string {
	return "souper"
}

func (ext *Souper) bankBase() uint32 {
	return uint32(ext.bank) << 14
}

func (ext *Souper) Access(write bool, address uint16, data uint8) (uint8, error) {
	if address < 0x4000 {
		return 0, nil
	}

	// "On reset, the 48KB area from $4000 - $FFFF available to cartridges is
	// arranged as follows :
	//
	// $4000 - $7FFF : 16KB Extended RAM
	// $8000 - $BFFF : 16KB Selectable ROM Bank
	// $C000 - $FFFF : 16KB Fixed ROM Bank
	//
	// This is mostly compatible with the Atari SuperCart layout if RAM Banking
	// is disabled."

	if address < 0x8000 {
		// "However, once RAM Banking is enabled by setting Souper Mode Bit 2 ($8003,2),
		// the 16KB RAM region is repartitioned :
		//
		// $4000 - $5FFF : 8KB Fixed Extended RAM
		// $6000 - $6FFF : 4KB Selectable V-Extended RAM
		// $7000 - $7FFF : 4KB Selectable D-Extended RAM"
		if ext.mode&0x04 == 0x04 {
			if address < 0x6000 {
				// 0x4000 to 0x5fff
				if write {
					ext.ram[address^0x4000] = data
				}
				return ext.ram[address^0x4000], nil
			} else if address < 0x7000 {
				// 0x6000 to 0x6fff
				if write {
					ext.ram[ext.vBank|uint32(address^0x6000)] = data
				}
				return ext.ram[ext.vBank|uint32(address^0x6000)], nil
			} else {
				// 0x7000 to 0x7fff
				if write {
					ext.ram[ext.dBank|uint32(address^0x7000)] = data
				}
				return ext.ram[ext.dBank|uint32(address^0x7000)], nil
			}
		} else {
			// 0x4000 to 0x7fff
			if write {
				ext.ram[address^0x4000] = data
			}
			return ext.ram[address^0x4000], nil
		}
	}

	// 0x8000 to 0xbfff
	if write && address < 0xc000 {
		// "A total of SIX mapping registers are available in the memory bastard, which
		// are accessed by writing to $8000 - $FFFF and repeat over an 8-Byte range.
		//
		// Software should access these registers only using $8000 - $8007 in case newer
		// variants of the mapper are developed with additional features.
		//
		// $0 = $8000 - $BFFF Bank Select, %xxxBBBBB
		// $1 = Character A Graphic Select, %BBBBBBBS
		// $2 = Character B Graphic Select, %BBBBBBBS
		// $3 = Souper Mode Enable, %xxxxxECS
		// $4 = $6000 - $6FFF EXRAM V-Bank Select, %xxxxxBBB
		// $5 = $7000 - $7FFF EXRAM D-Bank Select, %xxxxxBBB"
		switch address & 0x07 {
		case 0x0:
			ext.bank = int(data&0x1f) & (ext.numBanks - 1)
		case 0x1:
			ext.chrA = (uint32(data&0xfe) << 11) | (uint32(data&0x01) << 7)
		case 0x2:
			ext.chrB = (uint32(data&0xfe) << 11) | (uint32(data&0x01) << 7)
		case 0x3:
			ext.mode = data & 0x07
		case 0x4:
			ext.vBank = uint32(data&0x07) << 12
		case 0x5:
			ext.dBank = uint32(data&0x07) << 12
		case 0x7:
			// "There is a SEVENTH register which is a special case, it is used to alter the
			// state of the audio expansion communication port.
			//
			// Writing to $7 will alter the state of audCom and invert audReq_n to let the
			// audio processor know it has a command to read.
			//
			// Note that audReq_n is set up as an open drain output to simplify interfacing
			// with a 3.3V device if one is used as the audio expansion processor."
		}
		return 0, nil
	}

	// 0x8000 to 0xffff (non-write)

	// "Enabling both SOUPER Mode and Character Remapping through Souper Mode
	// Bits 0 & 1 ($8003,1 & 0), will allow additional Maria fetch trapping :
	//
	// - Fetches from $0000-$7FFF are unchanged
	// - Fetches from $8000-$9FFF are routed to the Fixed ROM Bank
	// - Fetches from $A000-$BFFF are routed to the Character A/B Bank Select
	// - Fetches from $C000-$FFFF are routed to EXRAM"
	if ext.hlt && ext.mode&0x03 == 0x03 {
		if address < 0xa000 {
			return ext.data[ext.mariaFixedBase|uint32(address^0x8000)], nil
		} else if address < 0xc000 {
			// "If it was Maria and in $A000 - $BFFF, our address will be generated based
			// upon our character bank select register and Maria's current read address
			// in the format: %00000BBB BBBBHHHH SLLLLLLL. This effectively splits the
			// region into two 2KB graphic data viewports which can be anywhere in ROM."
			if address&0x0080 != 0x0080 {
				a := ext.chrA | uint32(address&0x0f7f)
				return ext.data[a], nil
			} else {
				b := ext.chrB | uint32(address&0x0f7f)
				return ext.data[b], nil
			}
		}

		// addressing RAM for maria access is also sensitive to bit 2. this is similar to the
		// RAM selection logic above
		if ext.mode&0x04 == 0x04 {
			// 0xc000 to 0xffff
			if address < 0xe000 {
				return ext.ram[address^0xc000], nil
			} else if address < 0xf000 {
				return ext.ram[ext.vBank|uint32(address^0xe000)], nil
			}
			return ext.ram[ext.dBank|uint32(address^0xf000)], nil
		}
		return ext.ram[address^0xc000], nil
	}

	// "$8000 - $BFFF : 16KB Selectable ROM Bank
	// $C000 - $FFFF : 16KB Fixed ROM Bank"
	if address < 0xc000 {
		return ext.data[ext.bankBase()|uint32(address^0x8000)], nil
	}
	return ext.data[ext.sallyFixedBase|uint32(address^0x8000)], nil
}

func (ext *Souper) HLT(hlt bool) {
	ext.hlt = hlt
}
