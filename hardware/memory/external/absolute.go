package external

import (
	"fmt"
)

// Absolute implements the mapper used by the game F18 Hornet
// https://github.com/JetSetIlly/test7800/issues/30
type Absolute struct {
	// banks 2 and 3 are adjacent 32k of fixed data. they cannot be bank switched
	data [][]byte
	bank int
}

func NewAbsolute(_ Context, d []byte) (*Absolute, error) {
	ext := &Absolute{}

	const bankSize = 0x4000
	numBanks := len(d) / bankSize

	if numBanks != 4 {
		return nil, fmt.Errorf("absolute: mapper supports 2 16k banks + 32k of fixed data")
	}

	ext.data = make([][]byte, numBanks)

	// split data into banks
	for i := range len(ext.data) {
		o := bankSize * i
		ext.data[i] = d[o : o+bankSize]
	}
	ext.bank = 0

	return ext, nil
}

func (ext *Absolute) Label() string {
	return "Absolute"
}

func (ext *Absolute) Access(write bool, address uint16, data uint8) (uint8, error) {
	if address < 0x4000 {
		return 0, nil
	}

	if write {
		if address == 0x8000 {
			if data == 0x01 || data == 0x02 {
				ext.bank = int(data - 1)
			}
		}
		return 0, nil
	}

	if address < 0x8000 {
		return ext.data[ext.bank][address-0x4000], nil
	}

	if address < 0xc000 {
		// fixed 32k block, lower 16k
		return ext.data[2][address-0x8000], nil
	}

	// fixed 32k block, upper 16k
	return ext.data[3][address-0xc000], nil
}
