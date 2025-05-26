package external

type Flat struct {
	data   []byte
	origin uint16
}

func NewFlat(_ Context, d []byte) (*Flat, error) {
	return &Flat{
		data:   d,
		origin: uint16(0x10000 - len(d)),
	}, nil
}

func (ext *Flat) Label() string {
	return "Flat"
}

func (ext *Flat) Access(_ bool, address uint16, data uint8) (uint8, error) {
	if address < ext.origin {
		return 0, nil
	}
	return ext.data[address-ext.origin], nil
}
