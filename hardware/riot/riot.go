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

func (riot *RIOT) Access(write bool, idx uint16, data uint8) (uint8, error) {
	if write {
		return data, riot.Write(idx, data)
	}
	return riot.Read(idx)
}

func (riot *RIOT) Read(idx uint16) (uint8, error) {
	switch idx {
	case 0x00:
		return riot.swcha, nil
	case 0x01:
		// SWACNT
		return 0, nil
	case 0x02:
		// SWCHB
		// pro on by default (amateur would be 0x3f)
		return 0xff, nil
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

func (riot *RIOT) Write(idx uint16, data uint8) error {
	switch idx {
	case 0x00:
		riot.swcha = data
	case 0x01:
		// SWACNT
	case 0x02:
		// SWCHB
	case 0x03:
		// SWBCNT
	case 0x04, 0x10:
		// TIM1T
	case 0x05, 0x11:
		// TIM8T
	case 0x06, 0x12:
		// TIM64T
	case 0x07, 0x13:
		// T1024T
	}
	return nil
}
