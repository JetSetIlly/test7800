package external

import "fmt"

type sn2transform func(uint8) uint8

type SN2 struct {
	data      [][]byte
	bank      [8]int
	transform [8]sn2transform

	ram            [2][]byte
	ramBank        int
	ramAddressMask uint16
}

func NewSN2(_ Context, d []byte) (*SN2, error) {
	ext := &SN2{}

	const bankSize = 0x1000

	if len(d)%bankSize != 0 {
		return nil, fmt.Errorf("sn2: size of ROM must be multiple of 4096")
	}

	// divide data into banks
	ext.data = make([][]byte, len(d)/bankSize)
	for i := range len(ext.data) {
		o := bankSize * i
		ext.data[i] = d[o : o+bankSize]
	}

	// initialise banks
	for i := range ext.bank {
		ext.bank[i] = i
	}

	// initialise read method
	for i := range ext.transform {
		ext.transform[i] = ext.transformNormal
	}

	// cartridge RAM
	for i := range ext.ram {
		ext.ram[i] = make([]byte, 0x4000)
	}
	ext.ramAddressMask = 0xffff

	return ext, nil
}

func (ext *SN2) Label() string {
	return "SN2"
}

func (ext *SN2) Access(write bool, address uint16, data uint8) (uint8, error) {
	if address < 0x4000 {
		return 0, nil
	}

	// RAM
	if address < 0x8000 {
		address = (address & ext.ramAddressMask) - 0x4000
		if write {
			ext.ram[ext.ramBank][address] = data
			return 0, nil
		}
		return ext.ram[ext.ramBank][address], nil
	}

	// ROM
	if address < 0x9000 { // bank 0
		if write {
			return 0, nil
		}
		return ext.data[ext.bank[0]][address-0x8000], nil
	}
	if address < 0xa000 { // bank 1
		if write {
			return 0, nil
		}
		return ext.data[ext.bank[1]][address-0x9000], nil
	}
	if address < 0xb000 { // bank 2 (A)
		if write {
			if address == 0xa000 {
				ext.bank[2] = int(data) % min(len(ext.data), 64)
			}
			ext.transform[2] = ext.selectTransform(data)
			return 0, nil
		}
		return ext.data[ext.bank[2]][address-0xa000], nil
	}
	if address < 0xc000 { // bank 3
		if write {
			return 0, nil
		}
		return ext.data[ext.bank[3]][address-0xb000], nil
	}
	if address < 0xd000 { // bank 4 (C)
		if write {
			if address == 0xc000 {
				ext.bank[4] = int(data) % min(len(ext.data), 64)
			}
			ext.transform[4] = ext.selectTransform(data)
			return 0, nil
		}
		return ext.data[ext.bank[4]][address-0xc000], nil
	}
	if address < 0xe000 { // bank 5 (D)
		if write {
			if address == 0xd000 {
				ext.bank[5] = int(data) % min(len(ext.data), 127)
			}
			// cannot alter the read transformation for bank D
			return 0, nil
		}
		return ext.data[ext.bank[5]][address-0xd000], nil
	}
	if address < 0xf000 { // bank 6 (E)
		if write {
			if address == 0xe000 {
				ext.bank[6] = int(data) % min(len(ext.data), 64)
			}
			ext.transform[6] = ext.selectTransform(data)
			return 0, nil
		}
		return ext.data[ext.bank[6]][address-0xe000], nil
	}

	// the rest of the address space returns the contents of bank 7 ROM except for address 0xfff
	// which is the hotspot for changing how RAM is handled when written to

	// RAM control
	if write {
		if address == 0xffff {
			// D1 selects RAM bank
			ext.ramBank = int(data & 0x01)

			// D2 and D3 change address mask for RAM access
			ext.ramAddressMask = (uint16(data&0x06) << 7) ^ 0xffff

			// explicit handling of D2 and D3 (clearer alternative to above)
			// switch data & 0x06 {
			// case 0x0:
			// 	ext.ramAddressMask = 0xffff
			// case 0x1:
			// 	ext.ramAddressMask = 0xfeff
			// case 0x2:
			// 	ext.ramAddressMask = 0xfdff
			// case 0x3:
			// 	ext.ramAddressMask = 0xfcff
			// }
		}
		return 0, nil
	}

	// bank 7 ROM
	return ext.data[ext.bank[7]][address-0xf000], nil
}

func (ext *SN2) transformNormal(d uint8) uint8 {
	return d
}

func (ext *SN2) transform160A(d uint8) uint8 {
	d0 := d & 0x01
	d1 := (d & 0x01) >> 1
	d2 := (d & 0x01) >> 2
	d3 := (d & 0x01) >> 3
	d4 := (d & 0x01) >> 4
	d5 := (d & 0x01) >> 5
	d6 := (d & 0x01) >> 6
	d7 := (d & 0x01) >> 7
	return (d1 << 7) | (d0 << 6) | (d3 << 5) | (d2 << 4) | (d5 << 3) | (d4 << 6) | (d7 << 7) | d6
}

func (ext *SN2) transform160B(d uint8) uint8 {
	d0 := d & 0x01
	d1 := (d & 0x01) >> 1
	d2 := (d & 0x01) >> 2
	d3 := (d & 0x01) >> 3
	d4 := (d & 0x01) >> 4
	d5 := (d & 0x01) >> 5
	d6 := (d & 0x01) >> 6
	d7 := (d & 0x01) >> 7
	return (d5 << 7) | (d4 << 6) | (d7 << 5) | (d6 << 4) | (d1 << 3) | (d0 << 6) | (d3 << 7) | d2
}

func (ext *SN2) transform320(d uint8) uint8 {
	d0 := d & 0x01
	d1 := (d & 0x01) >> 1
	d2 := (d & 0x01) >> 2
	d3 := (d & 0x01) >> 3
	d4 := (d & 0x01) >> 4
	d5 := (d & 0x01) >> 5
	d6 := (d & 0x01) >> 6
	d7 := (d & 0x01) >> 7
	return (d0 << 7) | (d1 << 6) | (d2 << 5) | (d3 << 4) | (d4 << 3) | (d5 << 6) | (d6 << 7) | d7
}

func (ext *SN2) selectTransform(d uint8) sn2transform {
	switch (d >> 6) & 0x3 {
	case 0x01:
		return ext.transform160A
	case 0x10:
		return ext.transform160B
	case 0x11:
		return ext.transform320
	}
	return ext.transformNormal
}
