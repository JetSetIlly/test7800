package external

import "fmt"

// https://7800.8bitdev.org/index.php/ATARI_7800_BANKSWITCHING_GUIDE
//
// "There are several different types of Atari's SuperGame
// bankswitching. It basically consists of 8 16KB banks (0-7)
// that can be mapped in at $8000-$bfff. Bank 7 always is fixed
// at $c000-$ffff. To map in a chosen bank into $8000-$bfff you
// write it's bank number (0-7) to any address between $8000-bfff."
type Super struct {
	data  [][]byte
	bank  int
	exrom []byte
	exram []byte
}

func NewSuper(_ Context, d []byte, exrom bool, exram bool) (*Super, error) {
	ext := &Super{}

	if exrom && exram {
		return nil, fmt.Errorf("super: cannot support extra ROM and extra RAM")
	}

	if exrom {
		ext.exrom = d[:0x4000]
		d = d[0x4000:]
	}

	if exram {
		ext.exram = make([]byte, 0x8000-0x4000)
	}

	const bankSize = 0x4000

	if len(d)%bankSize != 0 {
		return nil, fmt.Errorf("super: unexpected payload size: %#x", len(d))
	}

	ext.data = make([][]byte, len(d)/bankSize)
	for i := range len(ext.data) {
		o := bankSize * i
		ext.data[i] = d[o : o+bankSize]
	}
	ext.bank = len(ext.data) - 2

	return ext, nil
}

func (ext *Super) Label() string {
	return "Standard"
}

func (ext *Super) Access(write bool, address uint16, data uint8) (uint8, error) {
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
		return 0, nil
	}
	if address < 0x8000 {
		return 0, nil
	}
	if address < 0xc000 {
		if write {
			ext.bank = int(data) % len(ext.data)
			return 0, nil
		}
		return ext.data[ext.bank][address-0x8000], nil
	}
	return ext.data[len(ext.data)-1][address-0xc000], nil
}
