package external

import (
	"fmt"
)

// Activision implements the activision type mapper
//
// https://forums.atariage.com/topic/384647-what-means-the-rom-file-am-and-om-in-trebors-rom-pack/#comment-5719218
//
// note that the information on that page isn't quite right but it gives a basic idea of the format.
// the Access() function for this mapper describes correctly the layout of the AM variant. The OM
// variant can be transformed into the AM variant by swapping each 8k block for each bank
type Activision struct {
	data [][]byte
	bank int
}

func NewActivision(_ Context, d []byte, originalOrder bool) (*Activision, error) {
	ext := &Activision{}

	const bankSize = 0x4000
	numBanks := len(d) / bankSize

	if numBanks != 8 {
		return nil, fmt.Errorf("activision: mapper supports eight banks only")
	}

	ext.data = make([][]byte, numBanks)

	// split data into banks
	for i := range len(ext.data) {
		o := bankSize * i
		ext.data[i] = d[o : o+bankSize]
	}
	ext.bank = 0

	// set ordering of data to the "alternative mapping" as assumed by the Access() function
	if originalOrder {
		for b := range len(ext.data) {
			for i := range 0x2000 {
				swp := ext.data[b][i]
				ext.data[b][i] = ext.data[b][i+0x2000]
				ext.data[b][i+0x2000] = swp
			}
		}
	}

	return ext, nil
}

func (ext *Activision) Label() string {
	return "Activision"
}

func (ext *Activision) Access(write bool, address uint16, data uint8) (uint8, error) {
	if address < 0x4000 {
		return 0, nil
	}

	// the following addresses assume the use of "alternative mapping" or AM order. any other
	// dumping order should have been handled in the NewActivision() function

	if write {
		if address >= 0xff80 {
			ext.bank = int(address & 0x0007)
		}
		return 0, nil
	}

	if address < 0x6000 {
		// second 8k of bank 6
		return ext.data[6][address-0x2000], nil
	}
	if address < 0x8000 {
		// first 8k of bank 6
		return ext.data[6][address-0x6000], nil
	}
	if address < 0xa000 {
		// second 8k of bank 7
		return ext.data[7][address-0x6000], nil
	}
	if address < 0xe000 {
		// the selected bank
		return ext.data[ext.bank][address-0xa000], nil
	}

	// first 8k of bank 7
	return ext.data[7][address-0xe000], nil
}
