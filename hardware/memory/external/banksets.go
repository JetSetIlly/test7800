package external

import (
	"fmt"

	"github.com/jetsetilly/test7800/logger"
)

// https://7800.8bitdev.org/index.php/Bankset_Bankswitching
type Banksets struct {
	dataSally [][]byte
	dataMaria [][]byte
	data      *[][]byte

	// origin is valid/used only for non-supergame data (ie. only when len(*data) is exactly 1)
	origin uint16

	ramSally []byte
	ramMaria []byte
	ram      *[]byte

	bank int
	hlt  bool
}

func NewBanksets(_ Context, supergame bool, d []byte, ram bool) (*Banksets, error) {
	// cartridge data is split into sally and maria banksets. exactly one half to each set.
	sz := len(d) >> 1

	// sanity checks and corrections. the code may tolerate different sizes and combinations but for
	// simplicity we don't allow it and only support the combinations found in the "Bankset Test Suite v1"
	switch sz {
	case 32768:
		if supergame {
			logger.Log(logger.Allow, "banksets", "32k ROMS should not have the supergame flag set")
			supergame = false
		}
	case 49152:
		if supergame {
			logger.Log(logger.Allow, "banksets", "48k ROMS should not have the supergame flag set")
			supergame = false
		}
		if ram {
			logger.Log(logger.Allow, "banksets", "48k ROMS cannot have cartridge RAM")
			ram = false
		}
	case 53248:
		if supergame {
			logger.Log(logger.Allow, "banksets", "52k ROMS should not have the supergame flag set")
			supergame = false
		}
		if ram {
			logger.Log(logger.Allow, "banksets", "52k ROMS cannot have cartridge RAM")
			ram = false
		}
	case 131072:
		if !supergame {
			logger.Log(logger.Allow, "banksets", "128k ROMS should have the supergame flag set")
			supergame = true
		}
	default:
		return nil, fmt.Errorf("banksets: unsupported ROM size: %d", len(d))
	}

	ext := &Banksets{}

	var numBanks int
	var bankSize int

	if supergame {
		bankSize = 0x4000
		numBanks = sz / bankSize
		logger.Logf(logger.Allow, "banksets", "%d banks", numBanks)
	} else {
		bankSize = sz
		numBanks = 1
		ext.origin = uint16(0x10000 - sz)
		logger.Logf(logger.Allow, "banksets", "non-supergame")
	}

	if ram {
		logger.Log(logger.Allow, "banksets", "with cartridge RAM")
		ext.ramSally = make([]byte, 0x4000)
		ext.ramMaria = make([]byte, 0x4000)
	}

	// split data into banks for sally access
	ext.dataSally = make([][]byte, numBanks)
	for i := range numBanks {
		o := bankSize * i
		ext.dataSally[i] = d[o : o+bankSize]
	}

	// done with the first half of the cartridge data
	d = d[sz:]

	// split data into banks for maria access
	ext.dataMaria = make([][]byte, numBanks)
	for i := range numBanks {
		o := (bankSize * i)
		ext.dataMaria[i] = d[o : o+bankSize]
	}

	ext.data = &ext.dataSally
	ext.ram = &ext.ramSally

	ext.hlt = false
	ext.bank = 0

	return ext, nil
}

func (ext *Banksets) Label() string {
	return "Banksets"
}

func (ext *Banksets) Access(write bool, address uint16, data uint8) (uint8, error) {
	if write && ext.hlt {
		panic("MARIA should not be writing to memory")
	}

	// supergame bankset ROMs
	if len(*ext.data) > 1 {
		if address < 0x4000 {
			return 0, nil
		}

		// 0x4000 to 0x7fff
		if address < 0x8000 {
			if len(*ext.ram) > 0 {
				if write {
					(*ext.ram)[address%0x4000] = data
					return 0, nil
				}
				return (*ext.ram)[address%0x4000], nil
			}
			return (*ext.data)[len(*ext.data)-2][address-0x4000], nil
		}

		// 0x8000 to 0xbfff
		if address < 0xc000 {
			if write {
				// it's not clear how the write data is treated if the value is greater
				// than the number of banks
				ext.bank = int((data & 0x0f) % uint8(len(*ext.data)))
			}
			return (*ext.data)[ext.bank][address-0x8000], nil
		}

		// 0xc000 to 0xffff

		// "sally's writes to $C000-$FFFF will be redirected to Maria's chunk of RAM"
		if write && len(ext.ramMaria) > 0 {
			ext.ramMaria[(address % 0x4000)] = data
			return 0, nil
		}

		// return data from last bank for all addresses of 0xc000 and above
		return (*ext.data)[len(*ext.data)-1][address-0xc000], nil
	}

	// handle RAM for non-supergame ROMs
	if len(*ext.ram) > 0 {
		if address < 0x8000 {
			if write {
				(*ext.ram)[address%0x4000] = data
				return 0, nil
			}
			return (*ext.ram)[address%0x4000], nil
		}

		// "sally's writes to $C000-$FFFF will be redirected to Maria's chunk of RAM"
		if write {
			ext.ramMaria[(address % 0x4000)] = data
			return 0, nil
		}
	}

	if address < ext.origin {
		return 0, nil
	}
	return (*ext.data)[0][address-ext.origin], nil
}

func (ext *Banksets) HLT(hlt bool) {
	ext.hlt = hlt
	if hlt {
		ext.data = &ext.dataMaria
		ext.ram = &ext.ramMaria
	} else {
		ext.data = &ext.dataSally
		ext.ram = &ext.ramSally
	}
}
