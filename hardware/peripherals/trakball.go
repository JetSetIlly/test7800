package peripherals

import (
	"fmt"

	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware/riot"
	"github.com/jetsetilly/test7800/hardware/tia"
)

// https://www.atarimania.com/documents/Atari-CX22-Trakball-Field-Service-Manual.pdf
// https://atarimuseum.ctrl-alt-rees.com/ahs_archives/archives/archives-techdocs-7800.htm
type Trakball struct {
	portRight bool
	riot      RIOT
	tia       TIA
	mem       Memory

	riotShift uint8
	button    tia.Register

	x    int
	y    int
	xclk uint8
	yclk uint8
}

func NewTrakball(r RIOT, t PaddlesTIA, m Memory, portRight bool) *Trakball {
	tb := &Trakball{
		portRight: portRight,
		riot:      r,
		tia:       t,
		mem:       m,
	}
	if portRight {
		tb.riotShift = 4
		tb.button = tia.INPT5
	} else {
		tb.riotShift = 0
		tb.button = tia.INPT4
	}
	return tb
}

func (tb *Trakball) IsAnalogue() bool {
	return true
}

func (tb *Trakball) IsController() bool {
	return true
}

func (tb *Trakball) Reset() {
	tb.tia.PortWrite(tb.button, 0x80, 0x7f)
}

func (tb *Trakball) Unplug() {
	tb.tia.PortWrite(tb.button, 0x80, 0x7f)
}

func (tb *Trakball) Update(inp gui.Input) error {
	switch inp.Action {
	case gui.TrakballFire:
		if inp.Data.(bool) {
			tb.tia.PortWrite(tb.button, 0x00, 0x7f)
		} else {
			tb.tia.PortWrite(tb.button, 0x80, 0x7f)
		}
	case gui.TrakballMove:
		d, ok := inp.Data.(gui.TrakballMoveData)
		if !ok {
			return fmt.Errorf("trakball: illegal trakball move data")
		}

		// deliberately not accumulating deltas to prevent excessive buffering
		tb.x = d.DeltaX
		tb.y = d.DeltaY
	}

	return nil
}

func (tb *Trakball) Tick() {
	// restricting the updating of SWCHA to the same rate at which it is read by the game/program.
	// when compared to the hardware this is not entirely accurate (the trakball has no way of
	// knowing when memory is read) but it's a good way of making sure that the SWCHA updates don't
	// get lost between reads
	if !tb.mem.LastReadIsRIOT() {
		return
	}

	if tb.x == 0 && tb.y == 0 {
		return
	}

	if tb.x < 0 {
		tb.xclk = (tb.xclk ^ 0x20) & 0xe0
		tb.x++
	} else if tb.x > 0 {
		tb.xclk = (tb.xclk ^ 0x20) | 0x10
		tb.x--
	}

	if tb.y > 0 {
		tb.yclk = (tb.yclk ^ 0x80) | 0x40
		tb.y--
	} else if tb.y < 0 {
		tb.yclk = (tb.yclk ^ 0x80) & 0xb0
		tb.y++
	}

	mask := func(v uint8) uint8 {
		return ^(v >> tb.riotShift)
	}

	tb.riot.PortWrite(riot.SWCHA, (tb.xclk|tb.yclk)>>tb.riotShift, mask(0xf0))
}
