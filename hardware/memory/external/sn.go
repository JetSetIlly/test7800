package external

import (
	"fmt"
	"slices"
)

type snTransformData func(uint8) uint8
type snTransformAddress func(uint16) uint16

type snBank struct {
	data    *[]byte
	mix     snTransformData
	address snTransformAddress
}

const snMaxBanks = 8

type SN struct {
	version string

	// the current state of the rom banks
	bank [snMaxBanks]*snBank

	// the backing data for the banks
	data [][]byte

	// catridge ram. SN1 has 32k split into 2 16k blocks. SN2 has 64k split into 4 16k blocks.
	// an SN1 cartridge therefore only uses the first two indices
	ram            [4][]byte
	ramBank        int
	ramAddressMask uint16

	// SN2 ram can be placed in 0x8000 to 0xbfff
	ramHigh bool

	// the way rom banks are mixed (or transformed) is different with SN2
	mixSN2 bool
}

func (ext *SN) isSN2() bool {
	return ext.version == "SN2"
}

func NewSN(_ Context, d []byte, version string) (*SN, error) {
	if !slices.Contains([]string{"SN2", "SN1"}, version) {
		return nil, fmt.Errorf("sn: unsupported version of mapper (%s)", version)
	}

	ext := &SN{
		version: version,
	}

	const bankSize = 0x1000

	if len(d)%bankSize != 0 {
		return nil, fmt.Errorf("sn: size of ROM must be multiple of 4096")
	}

	// divide data into banks
	ext.data = make([][]byte, len(d)/bankSize)
	for i := range len(ext.data) {
		o := bankSize * i
		ext.data[i] = d[o : o+bankSize]
	}

	// initialise banks and transform method
	for i := range snMaxBanks {
		ext.bank[i] = &snBank{
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

func (ext *SN) Label() string {
	return ext.version
}

func (ext *SN) Access(write bool, address uint16, data uint8) (uint8, error) {
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

	// ROM (or RAM if ramHigh is enabled)
	if address < 0x9000 { // bank 0
		if ext.ramHigh {
			// we need to manipulate the selected ramBank slightly rather than use it directly. we
			// want to use either ramBank 1 or ramBank 3 even if the selected ramBank if 1 or 3. in
			// this case the range 0x8000 to 0xbfff is the same as the range 0x4000 to 0x7fff, which
			// is intentional
			bank := int(ext.ramBank&0xfe) + 1

			address = (address & ext.ramAddressMask) - 0x8000
			if write {
				ext.ram[bank][address] = data
				return 0, nil
			}
			return ext.ram[bank][address], nil
		} else {
			b := ext.bank[0]
			if write {
				if address == 0x8000 {
					var idx int
					if !ext.isSN2() || ext.mixSN2 {
						idx = int(data&0x3f) % min(len(ext.data), 64)
						b.mix, b.address = ext.selectTransform(data)
					} else {
						idx = int(data) % min(len(ext.data), 256)
					}
					b.data = &ext.data[idx]
				}
			}
			return (*b.data)[address-0x8000], nil
		}
	}
	if address < 0xa000 { // bank 1
		b := ext.bank[1]
		if write {
			if address == 0x9000 {
				var idx int
				if !ext.isSN2() || ext.mixSN2 {
					idx = int(data&0x3f) % min(len(ext.data), 64)
					b.mix, b.address = ext.selectTransform(data)
				} else {
					idx = int(data) % min(len(ext.data), 256)
				}
				b.data = &ext.data[idx]
			}
		}
		return (*b.data)[address-0x9000], nil
	}
	if address < 0xb000 { // bank 2 (A)
		b := ext.bank[2]
		if write {
			if address == 0xa000 {
				var idx int
				if !ext.isSN2() || ext.mixSN2 {
					idx = int(data&0x3f) % min(len(ext.data), 64)
					b.mix, b.address = ext.selectTransform(data)
				} else {
					idx = int(data) % min(len(ext.data), 256)
				}
				b.data = &ext.data[idx]
			}
			return 0, nil
		}
		return b.mix((*b.data)[b.address(address)-0xa000]), nil
	}
	if address < 0xc000 { // bank 3
		b := ext.bank[3]
		if write {
			if address == 0xb000 {
				var idx int
				if !ext.isSN2() || ext.mixSN2 {
					return 0, nil
				} else {
					idx = int(data) % min(len(ext.data), 256)
				}
				b.data = &ext.data[idx]
			}
			return 0, nil
		}
		return b.mix((*b.data)[address-0xb000]), nil
	}
	if address < 0xd000 { // bank 4 (C)
		b := ext.bank[4]
		if write {
			if address == 0xc000 {
				var idx int
				if !ext.isSN2() || ext.mixSN2 {
					idx = int(data&0x3f) % min(len(ext.data), 64)
					b.mix, b.address = ext.selectTransform(data)
				} else {
					idx = int(data) % min(len(ext.data), 256)
				}
				b.data = &ext.data[idx]
			}
			return 0, nil
		}
		return b.mix((*b.data)[b.address(address)-0xc000]), nil
	}
	if address < 0xe000 { // bank 5 (D)
		b := ext.bank[5]
		if write {
			if address == 0xd000 {
				var idx int
				if !ext.isSN2() || ext.mixSN2 {
					idx = int(data&0x7f) % min(len(ext.data), 128)
					if ext.isSN2() && ext.mixSN2 {
						// cannot alter the read transformation for bank D in SN1 but we can for SN2
						// if mixSN2 is enabled
						b.mix, b.address = ext.selectTransform(data)
					}
				} else {
					idx = int(data) % min(len(ext.data), 256)
				}
				b.data = &ext.data[idx]
			}
			return 0, nil
		}
		return b.mix((*b.data)[address-0xd000]), nil
	}
	if address < 0xf000 { // bank 6 (E)
		b := ext.bank[6]
		if write {
			if address == 0xe000 {
				var idx int
				if !ext.isSN2() || ext.mixSN2 {
					idx = int(data&0x3f) % min(len(ext.data), 64)
					b.mix, b.address = ext.selectTransform(data)
				} else {
					idx = int(data) % min(len(ext.data), 256)
				}
				b.data = &ext.data[idx]
			}
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

			// additional bits are used in SN2
			if ext.isSN2() {
				if data&0x20 == 0x20 {
					ext.ramBank += 2
				}
				ext.mixSN2 = data&0x40 == 0x40
				ext.ramHigh = data&0x80 == 0x80
			}
		}
		return 0, nil
	}

	// bank 7 ROM
	b := ext.bank[7]
	return (*b.data)[address-0xf000], nil
}

func (ext *SN) transformDataNormal(d uint8) uint8 {
	return d
}

func (ext *SN) transformData160A(d uint8) uint8 {
	if ext.isSN2() && !ext.mixSN2 {
		return d
	}
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

func (ext *SN) transformData160B(d uint8) uint8 {
	if ext.isSN2() && !ext.mixSN2 {
		return d
	}
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

func (ext *SN) transformData320(d uint8) uint8 {
	if ext.isSN2() && !ext.mixSN2 {
		return d
	}
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

func (ext *SN) transformAddressNormal(a uint16) uint16 {
	return a
}

func (ext *SN) transformAddressReverse(a uint16) uint16 {
	a = (a & 0xff00) | (a ^ 0x00ff)
	return a
}

func (ext *SN) selectTransform(d uint8) (snTransformData, snTransformAddress) {
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
