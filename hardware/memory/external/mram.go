package external

// MRAM is created when bit 0x0080 of the a78 cartridge type field is on. Example ROM is the
// prototype of Rescue on Fractalus. The name mRAM comes from the A7800 rom.cpp file which describes
// this type of cartridge as "no bankswitch + mRAM chip"
type MRAM struct {
	data   []byte
	origin uint16
	ram    []byte
}

func NewMRAM(_ Context, d []byte) (*MRAM, error) {
	return &MRAM{
		data:   d,
		origin: uint16(0x10000 - len(d)),
		ram:    make([]byte, 0x4000),
	}, nil
}

func (ext *MRAM) Label() string {
	return "mRAM"
}

func (ext *MRAM) Access(write bool, address uint16, data uint8) (uint8, error) {
	if address < 0x4000 {
		return 0, nil
	}

	if address < 0x8000 {
		address &= 0xfeff
		if write {
			ext.ram[address-0x4000] = data
			return 0, nil
		}
		return ext.ram[address-0x4000], nil
	}

	if address < ext.origin {
		return 0, nil
	}
	return ext.data[address-ext.origin], nil
}
