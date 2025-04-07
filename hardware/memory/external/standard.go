package external

type Standard struct {
	data   []byte
	origin uint16
}

func NewStandard(_ Context, d []byte) (*Standard, error) {
	return &Standard{
		data:   d,
		origin: uint16(0x10000 - len(d)),
	}, nil
}

func (dev *Standard) Label() string {
	return "Standard"
}

func (dev *Standard) Access(_ bool, address uint16, data uint8) (uint8, error) {
	if address < dev.origin {
		return 0, nil
	}
	return dev.data[address-dev.origin], nil
}
