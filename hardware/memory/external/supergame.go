package external

import (
	"fmt"
)

// https://7800.8bitdev.org/index.php/ATARI_7800_BANKSWITCHING_GUIDE
//
// "There are several different types of Atari's SuperGame
// bankswitching. It basically consists of 8 16KB banks (0-7)
// that can be mapped in at $8000-$bfff. Bank 7 always is fixed
// at $c000-$ffff. To map in a chosen bank into $8000-$bfff you
// write it's bank number (0-7) to any address between $8000-bfff."
type Supergame struct {
	data  [][]byte
	bank  int
	exrom []byte
	exram []byte
}

func NewSupergame(_ Context, d []byte, exrom bool, exram bool) (*Supergame, error) {
	ext := &Supergame{}

	if exrom && exram {
		return nil, fmt.Errorf("supergame: cannot support extra ROM and extra RAM")
	}

	if exrom {
		ext.exrom = d[:0x4000]
		d = d[0x4000:]
	}

	if exram {
		ext.exram = make([]byte, 0x4000)
	}

	const bankSize = 0x4000

	if len(d)%bankSize != 0 {
		return nil, fmt.Errorf("supergame: unexpected payload size: %#x", len(d))
	}

	// supergame should have eight banks
	numBanks := len(d) / bankSize
	if numBanks != 8 {
		return nil, fmt.Errorf("supergame: it's not normal for a supergame cartridge to have %d banks", len(ext.data))
	}

	ext.data = make([][]byte, numBanks)

	// split data into banks
	for i := range len(ext.data) {
		o := bankSize * i
		ext.data[i] = d[o : o+bankSize]
	}
	ext.bank = 6

	return ext, nil
}

func (ext *Supergame) Label() string {
	if len(ext.exram) > 0 {
		return "Supergame (extra ram)"
	}
	if len(ext.exrom) > 0 {
		return "Supergame (extra rom)"
	}
	return "Supergame"
}

func (ext *Supergame) Access(write bool, address uint16, data uint8) (uint8, error) {
	if address < 0x4000 {
		return 0, nil
	}

	if address < 0x8000 {
		if len(ext.exrom) > 0 {
			return ext.exrom[address-0x4000], nil
		}

		if len(ext.exram) > 0 {
			if write {
				ext.exram[address-0x4000] = data
				return 0, nil
			}
			return ext.exram[address-0x4000], nil
		}

		// return data from bank 6 if there is no exrom and no exram
		//
		// there is a bit in the a64 header that controls this but there is at least one example
		// (Ace of Aces) where it's not set but the game still expects bank 6 to be there
		return ext.data[6][address-0x4000], nil
	}

	if address < 0xc000 {
		if write {
			// it's not clear how the write data is treated if the value is greater
			// than the number of banks. masking the three LSBs seems sensible
			ext.bank = int(data & 0x07)
		}
		return ext.data[ext.bank][address-0x8000], nil
	}

	// return data from bank 7 for all addresses of 0xc000 and above
	return ext.data[7][address-0xc000], nil
}
