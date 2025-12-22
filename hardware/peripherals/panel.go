package peripherals

import "github.com/jetsetilly/test7800/gui"

type Panel struct {
	riot RIOT
}

func NewPanel(riot RIOT) *Panel {
	p := &Panel{
		riot: riot,
	}
	return p
}

func (p *Panel) Reset() {
}

func (p *Panel) Update(inp gui.Input) error {
	switch inp.Action {
	case gui.Select:
		if inp.Data.(bool) {
			p.riot.PortWrite(0x02, 0x00, 0xfd)
		} else {
			p.riot.PortWrite(0x02, 0x02, 0xfd)
		}
	case gui.Start:
		if inp.Data.(bool) {
			p.riot.PortWrite(0x02, 0x00, 0xfe)
		} else {
			p.riot.PortWrite(0x02, 0x01, 0xfe)
		}
	case gui.Pause:
		if inp.Data.(bool) {
			p.riot.PortWrite(0x02, 0x00, 0xf7)
		} else {
			p.riot.PortWrite(0x02, 0x08, 0xf7)
		}
	case gui.P0Pro:
		if inp.Data.(bool) {
			p.riot.PortWrite(0x02, 0x80, 0x7f)
		} else {
			p.riot.PortWrite(0x02, 0x00, 0x7f)
		}
	case gui.P1Pro:
		if inp.Data.(bool) {
			p.riot.PortWrite(0x02, 0x40, 0xbf)
		} else {
			p.riot.PortWrite(0x02, 0x00, 0xbf)
		}
	}

	return nil
}

func (p *Panel) Step() {
}
