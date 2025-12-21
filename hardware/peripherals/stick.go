package peripherals

import (
	"github.com/jetsetilly/test7800/gui"
)

type Stick struct {
	riot RIOT
	tia  TIA

	portRight bool

	riotShift uint16
	tiaButton uint16

	twoButtons bool
	tiaButtonA uint16
	tiaButtonB uint16
}

// https://forums.atariage.com/topic/127162-question-about-joysticks-and-how-they-are-read/#findComment-1537159
func NewStick(riot RIOT, tia TIA, portRight bool, twoButtons bool) *Stick {
	st := &Stick{
		riot:       riot,
		tia:        tia,
		portRight:  portRight,
		twoButtons: twoButtons,
	}

	if portRight {
		st.riotShift = 0x04
	}

	if twoButtons {
		if portRight {
			st.tiaButtonA = 0x0b // INPT3
			st.tiaButtonB = 0x0a // INPT2
		} else {
			st.tiaButtonA = 0x09 // INPT1
			st.tiaButtonB = 0x08 // INPT0
		}
	} else {
		if portRight {
			st.tiaButton = 0x0d // INPT5
			st.tiaButton = 0x0d
		} else {
			st.tiaButton = 0x0c // INPT4
			st.tiaButton = 0x0c
		}
	}

	st.Reset()

	return st
}

func (st *Stick) Reset() {
	if st.twoButtons {
		st.tia.PortWrite(st.tiaButtonA, 0x00, 0x7f)
		st.tia.PortWrite(st.tiaButtonB, 0x00, 0x7f)
	} else {
		st.tia.PortWrite(st.tiaButton, 0x80, 0x7f)
	}
}

func (st *Stick) Update(inp gui.Input) error {
	mask := func(v uint8) uint8 {
		return ^(v >> st.riotShift)
	}

	switch inp.Action {
	case gui.StickLeft:
		if inp.Data.(bool) {
			// unset the opposite direction first (applies to all other directions below)
			st.riot.PortWrite(0x00, 0x80>>st.riotShift, mask(0x80))
			st.riot.PortWrite(0x00, 0x00>>st.riotShift, mask(0x40))
		} else {
			st.riot.PortWrite(0x00, 0x40>>st.riotShift, mask(0x40))
		}
	case gui.StickUp:
		if inp.Data.(bool) {
			st.riot.PortWrite(0x00, 0x20>>st.riotShift, mask(0x20))
			st.riot.PortWrite(0x00, 0x00>>st.riotShift, mask(0x10))
		} else {
			st.riot.PortWrite(0x00, 0x10>>st.riotShift, mask(0x10))
		}
	case gui.StickRight:
		if inp.Data.(bool) {
			st.riot.PortWrite(0x00, 0x40>>st.riotShift, mask(0x40))
			st.riot.PortWrite(0x00, 0x00>>st.riotShift, mask(0x80))
		} else {
			st.riot.PortWrite(0x00, 0x80>>st.riotShift, mask(0x80))
		}
	case gui.StickDown:
		if inp.Data.(bool) {
			st.riot.PortWrite(0x00, 0x10>>st.riotShift, mask(0x10))
			st.riot.PortWrite(0x00, 0x00>>st.riotShift, mask(0x20))
		} else {
			st.riot.PortWrite(0x00, 0x20>>st.riotShift, mask(0x20))
		}
	case gui.StickButtonA:
		if st.twoButtons {
			// the two-button stick fire buttons have opposite logic to single button stick
			if inp.Data.(bool) {
				st.tia.PortWrite(st.tiaButtonA, 0x80, 0x7f)
			} else {
				st.tia.PortWrite(st.tiaButtonA, 0x00, 0x7f)
			}
		} else {
			if inp.Data.(bool) {
				st.tia.PortWrite(st.tiaButton, 0x00, 0x7f)
			} else {
				st.tia.PortWrite(st.tiaButton, 0x80, 0x7f)
			}
		}
	case gui.StickButtonB:
		if st.twoButtons {
			// the two-button stick fire buttons have opposite logic to single button stick
			if inp.Data.(bool) {
				st.tia.PortWrite(st.tiaButtonB, 0x80, 0x7f)
			} else {
				st.tia.PortWrite(st.tiaButtonB, 0x00, 0x7f)
			}
		} else {
			if inp.Data.(bool) {
				st.tia.PortWrite(st.tiaButton, 0x00, 0x7f)
			} else {
				st.tia.PortWrite(st.tiaButton, 0x80, 0x7f)
			}
		}
	}

	return nil
}

func (st *Stick) Step() {
}
