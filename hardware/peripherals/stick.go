package peripherals

import (
	"fmt"

	"github.com/jetsetilly/test7800/gui"
)

type Stick struct {
	riot RIOT
	tia  TIA

	riotShift int

	portRight  bool
	twoButtons bool

	buttonA uint16
	buttonB uint16
	button  uint16

	// https://forums.atariage.com/topic/127162-question-about-joysticks-and-how-they-are-read/#findComment-1537159
	singleMask uint8
}

func NewStick(riot RIOT, tia TIA, portRight bool, twoButtons bool) *Stick {
	st := &Stick{
		riot:       riot,
		tia:        tia,
		portRight:  portRight,
		twoButtons: twoButtons,
	}
	if portRight {
		st.riotShift = 4
		st.buttonA = 0x0b
		st.buttonB = 0x0a
		st.button = 0x0d
		st.singleMask = 0x01
	} else {
		st.buttonA = 0x09
		st.buttonB = 0x08
		st.button = 0x0c
		st.singleMask = 0x04
	}
	st.Reset()
	return st
}

func (st *Stick) Reset() {
	if st.twoButtons {
		st.tia.PortWrite(st.buttonA, 0x00, 0x7f)
		st.tia.PortWrite(st.buttonB, 0x00, 0x7f)
	} else {
		st.tia.PortWrite(st.button, 0x80, 0x7f)
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
		b, err := st.riot.Read(0x02)
		if err != nil {
			return fmt.Errorf("stick button a: %w", err)
		}
		if b&st.singleMask == st.singleMask {
			if inp.Data.(bool) {
				st.tia.PortWrite(st.button, 0x00, 0x7f)
			} else {
				st.tia.PortWrite(st.button, 0x80, 0x7f)
			}
		} else {
			// the two-button stick write to INPT0/INPT1 has an opposite logic to
			// the write to INPT4/INPT5
			if inp.Data.(bool) {
				st.tia.PortWrite(st.buttonA, 0x80, 0x7f)
			} else {
				st.tia.PortWrite(st.buttonA, 0x00, 0x7f)
			}
		}
	case gui.StickButtonB:
		b, err := st.riot.Read(0x02)
		if err != nil {
			return fmt.Errorf("stick button b: %w", err)
		}
		if b&st.singleMask == st.singleMask {
			if inp.Data.(bool) {
				st.tia.PortWrite(st.button, 0x00, 0x7f)
			} else {
				st.tia.PortWrite(st.button, 0x80, 0x7f)
			}
		} else {
			// the two-button stick write to INPT0/INPT1 has an opposite logic to
			// the write to INPT4/INPT5
			if inp.Data.(bool) {
				st.tia.PortWrite(st.buttonB, 0x80, 0x7f)
			} else {
				st.tia.PortWrite(st.buttonB, 0x00, 0x7f)
			}
		}
	}

	return nil
}

func (st *Stick) Step() {
}
