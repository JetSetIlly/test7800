package peripherals

import (
	"fmt"

	"github.com/jetsetilly/test7800/gui"
)

type Stick struct {
	riot RIOT
	tia  TIA

	portRight  bool
	twoButtons bool
}

// https://forums.atariage.com/topic/127162-question-about-joysticks-and-how-they-are-read/#findComment-1537159
func NewStick(riot RIOT, tia TIA, portRight bool, twoButtons bool) *Stick {
	st := &Stick{
		riot:       riot,
		tia:        tia,
		portRight:  portRight,
		twoButtons: twoButtons,
	}
	st.Reset()
	return st
}

func (st *Stick) Reset() {
	if st.twoButtons {
		st.tia.PortWrite(0x09, 0x00, 0x7f)
		st.tia.PortWrite(0x08, 0x00, 0x7f)
	} else {
		st.tia.PortWrite(0x0c, 0x80, 0x7f)
	}
}

func (st *Stick) Update(inp gui.Input) error {
	switch inp.Action {
	case gui.StickLeft:
		if inp.Data.(bool) {
			// unset the opposite direction first (applies to all
			// other directions below)
			st.riot.PortWrite(0x00, 0x80, 0x7f)
			st.riot.PortWrite(0x00, 0x00, 0xbf)
		} else {
			st.riot.PortWrite(0x00, 0x40, 0xbf)
		}
	case gui.StickUp:
		if inp.Data.(bool) {
			st.riot.PortWrite(0x00, 0x20, 0xdf)
			st.riot.PortWrite(0x00, 0x00, 0xef)
		} else {
			st.riot.PortWrite(0x00, 0x10, 0xef)
		}
	case gui.StickRight:
		if inp.Data.(bool) {
			st.riot.PortWrite(0x00, 0x40, 0xbf)
			st.riot.PortWrite(0x00, 0x00, 0x7f)
		} else {
			st.riot.PortWrite(0x00, 0x80, 0x7f)
		}
	case gui.StickDown:
		if inp.Data.(bool) {
			st.riot.PortWrite(0x00, 0x10, 0xef)
			st.riot.PortWrite(0x00, 0x00, 0xdf)
		} else {
			st.riot.PortWrite(0x00, 0x20, 0xdf)
		}
	case gui.StickButtonA:
		// https://forums.atariage.com/topic/127162-question-about-joysticks-and-how-they-are-read/#findComment-1537159
		b, err := st.riot.Read(0x02)
		if err != nil {
			return fmt.Errorf("stick button a: %w", err)
		}
		if b&0x04 == 0x04 {
			if inp.Data.(bool) {
				st.tia.PortWrite(0x0c, 0x00, 0x7f)
			} else {
				st.tia.PortWrite(0x0c, 0x80, 0x7f)
			}
		} else {
			// the two-button stick write to INPT0/INPT1 has an opposite logic to
			// the write to INPT4/INPT5
			if inp.Data.(bool) {
				st.tia.PortWrite(0x09, 0x80, 0x7f)
			} else {
				st.tia.PortWrite(0x09, 0x00, 0x7f)
			}
		}
	case gui.StickButtonB:
		// https://forums.atariage.com/topic/127162-question-about-joysticks-and-how-they-are-read/#findComment-1537159
		b, err := st.riot.Read(0x02)
		if err != nil {
			return fmt.Errorf("stick button b: %w", err)
		}
		if b&0x04 == 0x04 {
			if inp.Data.(bool) {
				st.tia.PortWrite(0x0c, 0x00, 0x7f)
			} else {
				st.tia.PortWrite(0x0c, 0x80, 0x7f)
			}
		} else {
			// the two-button stick write to INPT0/INPT1 has an opposite logic to
			// the write to INPT4/INPT5
			if inp.Data.(bool) {
				st.tia.PortWrite(0x08, 0x80, 0x7f)
			} else {
				st.tia.PortWrite(0x08, 0x00, 0x7f)
			}
		}
	}

	return nil
}

func (st *Stick) Step() {
}
