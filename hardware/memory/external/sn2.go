package external

import "fmt"

type sn2transformData func(uint8) uint8
type sn2transformAddress func(uint16) uint16

type sn2Bank struct {
	data    *[]byte
	mix     sn2transformData
	address sn2transformAddress
}

const sn2MaxBanks = 8

type SN2 struct {
	// the current state of the rom banks
	bank [sn2MaxBanks]*sn2Bank

	// the backing data for the banks
	data [][]byte

	// catridge ram
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

	// initialise banks and transform method
	for i := range sn2MaxBanks {
		ext.bank[i] = &sn2Bank{
			data:    &ext.data[i],
			mix:     ext.transformDataNormal,
			address: ext.transformAddressNormal,
		}
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
		b := ext.bank[0]
		return (*b.data)[address-0x8000], nil
	}
	if address < 0xa000 { // bank 1
		if write {
			return 0, nil
		}
		b := ext.bank[1]
		return (*b.data)[address-0x9000], nil
	}
	if address < 0xb000 { // bank 2 (A)
		b := ext.bank[2]
		if write {
			if address == 0xa000 {
				idx := int(data&0x3f) % min(len(ext.data), 64)
				b.data = &ext.data[idx]
			}
			b.mix, b.address = ext.selectTransform(data)
			return 0, nil
		}
		return b.mix((*b.data)[b.address(address)-0xa000]), nil
	}
	if address < 0xc000 { // bank 3
		if write {
			return 0, nil
		}
		b := ext.bank[3]
		return (*b.data)[address-0xb000], nil
	}
	if address < 0xd000 { // bank 4 (C)
		b := ext.bank[4]
		if write {
			if address == 0xc000 {
				idx := int(data&0x3f) % min(len(ext.data), 64)
				b.data = &ext.data[idx]
			}
			b.mix, b.address = ext.selectTransform(data)
			return 0, nil
		}
		return b.mix((*b.data)[b.address(address)-0xc000]), nil
	}
	if address < 0xe000 { // bank 5 (D)
		b := ext.bank[5]
		if write {
			if address == 0xd000 {
				idx := int(data&0x7f) % min(len(ext.data), 128)
				b.data = &ext.data[idx]
			}
			// cannot alter the read transformation for bank D
			return 0, nil
		}
		return (*b.data)[address-0xd000], nil
	}
	if address < 0xf000 { // bank 6 (E)
		b := ext.bank[6]
		if write {
			if address == 0xe000 {
				idx := int(data&0x3f) % min(len(ext.data), 64)
				b.data = &ext.data[idx]
			}
			b.mix, b.address = ext.selectTransform(data)
			return 0, nil
		}
		return b.mix((*b.data)[b.address(address)-0xe000]), nil
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
	b := ext.bank[7]
	return (*b.data)[address-0xf000], nil
}

func (ext *SN2) transformDataNormal(d uint8) uint8 {
	return d
}

func (ext *SN2) transformData160A(d uint8) uint8 {
	d0 := d & 0x01
	d1 := (d & 0x02) >> 1
	d2 := (d & 0x04) >> 2
	d3 := (d & 0x08) >> 3
	d4 := (d & 0x10) >> 4
	d5 := (d & 0x20) >> 5
	d6 := (d & 0x40) >> 6
	d7 := (d & 0x80) >> 7
	return (d1 << 7) | (d0 << 6) | (d3 << 5) | (d2 << 4) | (d5 << 3) | (d4 << 2) | (d7 << 1) | d6
}

func (ext *SN2) transformData160B(d uint8) uint8 {
	d0 := d & 0x01
	d1 := (d & 0x02) >> 1
	d2 := (d & 0x04) >> 2
	d3 := (d & 0x08) >> 3
	d4 := (d & 0x10) >> 4
	d5 := (d & 0x20) >> 5
	d6 := (d & 0x40) >> 6
	d7 := (d & 0x80) >> 7
	return (d5 << 7) | (d4 << 6) | (d7 << 5) | (d6 << 4) | (d1 << 3) | (d0 << 2) | (d3 << 1) | d2
}

func (ext *SN2) transformData320(d uint8) uint8 {
	d0 := d & 0x01
	d1 := (d & 0x02) >> 1
	d2 := (d & 0x04) >> 2
	d3 := (d & 0x08) >> 3
	d4 := (d & 0x10) >> 4
	d5 := (d & 0x20) >> 5
	d6 := (d & 0x40) >> 6
	d7 := (d & 0x80) >> 7
	return (d0 << 7) | (d1 << 6) | (d2 << 5) | (d3 << 4) | (d4 << 3) | (d5 << 2) | (d6 << 1) | d7
}

func (ext *SN2) transformAddressNormal(a uint16) uint16 {
	return a
}

func (ext *SN2) transformAddressReverse(a uint16) uint16 {
	a = (a & 0xff00) | (a ^ 0x00ff)
	return a
}

func (ext *SN2) selectTransform(d uint8) (sn2transformData, sn2transformAddress) {
	switch (d >> 6) & 0x03 {
	case 0b01:
		return ext.transformData160A, ext.transformAddressReverse
	case 0b10:
		return ext.transformData160B, ext.transformAddressReverse
	case 0b11:
		return ext.transformData320, ext.transformAddressReverse
	}
	return ext.transformDataNormal, ext.transformAddressNormal
}
