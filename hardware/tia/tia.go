package tia

type TIA struct {
	mem  Memory
	inpt [6]uint8
}

type Memory interface {
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
}

func Create(mem Memory) *TIA {
	return &TIA{
		mem: mem,

		// inpt initialised as though sticks are being used
		inpt: [6]uint8{
			0x00, 0x00, 0x00, 0x00,
			0x80, 0x80,
		},
	}
}

func (tia *TIA) Label() string {
	return "TIA"
}

func (tia *TIA) Status() string {
	return tia.Label()
}

func (tia *TIA) Read(address uint16) (uint8, error) {
	switch address {
	case 0x08:
		return tia.inpt[0], nil
	case 0x09:
		return tia.inpt[1], nil
	case 0x0a:
		return tia.inpt[2], nil
	case 0x0b:
		return tia.inpt[3], nil
	case 0x0c:
		return tia.inpt[4], nil
	case 0x0d:
		return tia.inpt[5], nil
	}
	return 0, nil
}

func (tia *TIA) Write(address uint16, data uint8) error {
	switch address {
	case 0x08:
		tia.inpt[0] = data
	case 0x09:
		tia.inpt[1] = data
	case 0x0a:
		tia.inpt[2] = data
	case 0x0b:
		tia.inpt[3] = data
	case 0x0c:
		tia.inpt[4] = data
	case 0x0d:
		tia.inpt[5] = data
	}
	return nil
}
