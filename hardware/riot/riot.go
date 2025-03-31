package riot

type RIOT struct {
	mem   Memory
	swcha uint8
}

type Memory interface {
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
}

func Create(mem Memory) *RIOT {
	return &RIOT{
		mem: mem,

		// swcha initialised as though stick is being used
		swcha: 0xff,
	}
}

func (riot *RIOT) Label() string {
	return "RIOT"
}

func (riot *RIOT) Status() string {
	return riot.Label()
}

func (riot *RIOT) Read(address uint16) (uint8, error) {
	switch address {
	case 0x00:
		return riot.swcha, nil
	case 0x01:
		// SWACNT
		return 0, nil
	case 0x02:
		// SWCHB
		return 0x3f, nil
	case 0x03:
		// SWBCNT
		return 0, nil
	case 0x04:
		// INTIM
		return 0, nil
	case 0x05:
		// TIMINT
		return 0, nil
	}
	return 0, nil
}

func (riot *RIOT) Write(address uint16, data uint8) error {
	switch address {
	case 0x00:
		riot.swcha = data
	case 0x01:
	case 0x02:
	case 0x03:
	case 0x04:
	case 0x05:
	}
	return nil
}
