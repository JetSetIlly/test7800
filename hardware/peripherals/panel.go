package peripherals

import (
	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware/riot"
)

type Panel struct {
	riot RIOT
}

func NewPanel(r RIOT) *Panel {
	p := &Panel{
		riot: r,
	}
	return p
}

func (p *Panel) IsAnalogue() bool {
	return false
}

func (p *Panel) Reset() {
}

func (p *Panel) Unplug() {
	p.riot.PortWrite(riot.SWCHB, 0x00, 0x00)
}

func (p *Panel) Update(inp gui.Input) error {
	switch inp.Action {
	case gui.Select:
		if inp.Data.(bool) {
			p.riot.PortWrite(riot.SWCHB, 0x00, 0xfd)
		} else {
			p.riot.PortWrite(riot.SWCHB, 0x02, 0xfd)
		}
	case gui.Start:
		if inp.Data.(bool) {
			p.riot.PortWrite(riot.SWCHB, 0x00, 0xfe)
		} else {
			p.riot.PortWrite(riot.SWCHB, 0x01, 0xfe)
		}
	case gui.Pause:
		if inp.Data.(bool) {
			p.riot.PortWrite(riot.SWCHB, 0x00, 0xf7)
		} else {
			p.riot.PortWrite(riot.SWCHB, 0x08, 0xf7)
		}
	case gui.P0Pro:
		if inp.Data.(bool) {
			p.riot.PortWrite(riot.SWCHB, 0x80, 0x7f)
		} else {
			p.riot.PortWrite(riot.SWCHB, 0x00, 0x7f)
		}
	case gui.P1Pro:
		if inp.Data.(bool) {
			p.riot.PortWrite(riot.SWCHB, 0x40, 0xbf)
		} else {
			p.riot.PortWrite(riot.SWCHB, 0x00, 0xbf)
		}
	}

	return nil
}

func (p *Panel) Tick() {
}
