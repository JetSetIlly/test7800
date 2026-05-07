package peripherals

import (
	"fmt"

	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware/riot"
	"github.com/jetsetilly/test7800/hardware/tia"
)

type Stick struct {
	riot RIOT
	tia  TIA

	// the amount which RIOT/SWCHA values are shifted by depending on the port the stick is attached
	// to the mask value has been pre-shifted and does not need to be shifted
	riotShift uint8
	riotMask  uint8

	portRight  bool
	twoButtons bool

	buttonA tia.Register
	buttonB tia.Register
	button  tia.Register

	// https://forums.atariage.com/topic/127162-question-about-joysticks-and-how-they-are-read/#findComment-1537159
	singleMask uint8

	// current state of the SWCHA register and the button registers
	//
	// these fields are used to faciliate quadtari compatability. the quadtari only supports single
	// button joysticks which iw why we only track the singleButton configuration (because the
	// quadtari does not support multi-button sticks)
	swcha            uint8
	singleButtonFire bool

	// match inputSource with incoming gui.Input from Update() function
	inputSource gui.InputSource
}

func NewStick(r RIOT, t TIA, portRight bool, twoButtons bool) *Stick {
	st := &Stick{
		riot:       r,
		tia:        t,
		portRight:  portRight,
		twoButtons: twoButtons,
		singleMask: 0x04,
		swcha:      0xf0,
	}

	if portRight {
		st.buttonA = tia.INPT3
		st.buttonB = tia.INPT2
		st.button = tia.INPT5
		st.riotShift = 4
		st.riotMask = 0xf0
	} else {
		st.buttonA = tia.INPT1
		st.buttonB = tia.INPT0
		st.button = tia.INPT4
		st.riotShift = 0
		st.riotMask = 0x0f
	}

	return st
}

func (st *Stick) IsAnalogue() bool {
	return false
}

func (st *Stick) IsController() bool {
	return true
}

func (st *Stick) Reset() {
	if st.twoButtons {
		st.tia.PortWrite(st.buttonA, 0x00, 0x7f)
		st.tia.PortWrite(st.buttonB, 0x00, 0x7f)
	}
	st.tia.PortWrite(st.button, 0x80, 0x7f)
}

func (st *Stick) Unplug() {
	if st.twoButtons {
		st.tia.PortWrite(st.buttonA, 0x00, 0x7f)
		st.tia.PortWrite(st.buttonB, 0x00, 0x7f)
	}
	st.tia.PortWrite(st.button, 0x00, 0x7f)
}

func (st *Stick) Update(inp gui.Input) error {
	if st.inputSource.Type == gui.InputAny ||
		(inp.Source.Type == st.inputSource.Type && inp.Source.Position == st.inputSource.Position) {

		switch inp.Action {
		case gui.StickLeft:
			if inp.Data.(bool) {
				if st.swcha&0x40 != 0x00 {
					st.swcha |= 0x80
					st.swcha ^= 0x40
					st.riot.PortWrite(riot.SWCHA, st.swcha>>st.riotShift, st.riotMask)
				}
			} else {
				if st.swcha&0x40 == 0x00 {
					st.swcha |= 0x40
					st.riot.PortWrite(riot.SWCHA, st.swcha>>st.riotShift, st.riotMask)
				}
			}
		case gui.StickUp:
			if inp.Data.(bool) {
				if st.swcha&0x10 != 0x00 {
					st.swcha |= 0x20
					st.swcha ^= 0x10
					st.riot.PortWrite(riot.SWCHA, st.swcha>>st.riotShift, st.riotMask)
				}
			} else {
				if st.swcha&0x10 == 0x00 {
					st.swcha |= 0x10
					st.riot.PortWrite(riot.SWCHA, st.swcha>>st.riotShift, st.riotMask)
				}
			}
		case gui.StickRight:
			if inp.Data.(bool) {
				if st.swcha&0x80 != 0x00 {
					st.swcha |= 0x40
					st.swcha ^= 0x80
					st.riot.PortWrite(riot.SWCHA, st.swcha>>st.riotShift, st.riotMask)
				}
			} else {
				if st.swcha&0x80 == 0x00 {
					st.swcha |= 0x80
					st.riot.PortWrite(riot.SWCHA, st.swcha>>st.riotShift, st.riotMask)
				}
			}
		case gui.StickDown:
			if inp.Data.(bool) {
				if st.swcha&0x20 != 0x00 {
					st.swcha |= 0x10
					st.swcha ^= 0x20
					st.riot.PortWrite(riot.SWCHA, st.swcha>>st.riotShift, st.riotMask)
				}
			} else {
				if st.swcha&0x20 == 0x00 {
					st.swcha |= 0x20
					st.riot.PortWrite(riot.SWCHA, st.swcha>>st.riotShift, st.riotMask)
				}
			}
		case gui.StickButtonA:
			b, err := st.riot.PortRead(riot.SWCHB)
			if err != nil {
				return fmt.Errorf("stick button a: %w", err)
			}
			if b&st.singleMask == st.singleMask {
				if inp.Data.(bool) {
					st.tia.PortWrite(st.button, 0x00, 0x7f)
					st.singleButtonFire = true
				} else {
					st.tia.PortWrite(st.button, 0x80, 0x7f)
					st.singleButtonFire = false
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
			b, err := st.riot.PortRead(riot.SWCHB)
			if err != nil {
				return fmt.Errorf("stick button b: %w", err)
			}
			if b&st.singleMask == st.singleMask {
				if inp.Data.(bool) {
					st.tia.PortWrite(st.button, 0x00, 0x7f)
					st.singleButtonFire = true
				} else {
					st.tia.PortWrite(st.button, 0x80, 0x7f)
					st.singleButtonFire = false
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
	}

	return nil
}

func (st *Stick) Tick() {
}

func (st *Stick) SWCHA() (uint8, uint8) {
	return st.swcha >> st.riotShift, st.riotMask
}

func (st *Stick) Button() (tia.Register, bool) {
	return st.button, st.singleButtonFire
}

func (st *Stick) SetInputFilter(inputSource gui.InputSource, primary bool) {
	if primary {
		st.inputSource = inputSource
	}
}
