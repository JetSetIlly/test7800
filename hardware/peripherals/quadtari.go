package peripherals

import (
	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware/riot"
	"github.com/jetsetilly/test7800/hardware/tia"
)

type quadtari interface {
	IsAnalogue() bool
	IsController() bool
	Reset()
	Unplug()
	Update(inp gui.Input) error
	Tick()
	SWCHA() (uint8, uint8)
	Button() (tia.Register, bool)
}

type Quadtari struct {
	riot RIOT
	tia  GroundedTIA
	A    quadtari
	B    quadtari

	// whether the TIA returned grounded on the most recent tick
	grounded bool
}

func NewQuadtari(r RIOT, tia GroundedTIA) *Quadtari {
	return &Quadtari{
		riot: r,
		tia:  tia,
	}
}

func (q *Quadtari) IsAnalogue() bool {
	if q.tia.Grounded() {
		return q.A.IsAnalogue()
	}
	return q.B.IsAnalogue()
}

func (q *Quadtari) IsController() bool {
	if q.tia.Grounded() {
		return q.A.IsController()
	}
	return q.B.IsController()
}

func (q *Quadtari) Reset() {
	q.A.Reset()
	q.B.Reset()
	q.tia.PortWrite(tia.INPT0, 0x00, 0x7f)
	q.tia.PortWrite(tia.INPT1, 0x80, 0x7f)
}

func (q *Quadtari) Unplug() {
	q.A.Unplug()
	q.B.Unplug()
	q.tia.PortWrite(tia.INPT0, 0x00, 0x7f)
	q.tia.PortWrite(tia.INPT1, 0x00, 0x7f)
}

func (q *Quadtari) Update(inp gui.Input) error {
	switch inp.Source {
	case "keyboard":
		return q.A.Update(inp)
	case "gamepad":
		return q.B.Update(inp)
	}
	return nil
}

func (q *Quadtari) Tick() {
	q.A.Tick()
	q.B.Tick()
	grounded := q.tia.Grounded()
	if grounded != q.grounded {
		if grounded {
			data, mask := q.A.SWCHA()
			q.riot.PortWrite(riot.SWCHA, data, mask)
			if reg, fire := q.A.Button(); fire {
				q.tia.PortWrite(reg, 0x00, 0x7f)
			} else {
				q.tia.PortWrite(reg, 0x80, 0x7f)
			}
		} else {
			data, mask := q.B.SWCHA()
			q.riot.PortWrite(riot.SWCHA, data, mask)
			if reg, fire := q.B.Button(); fire {
				q.tia.PortWrite(reg, 0x00, 0x7f)
			} else {
				q.tia.PortWrite(reg, 0x80, 0x7f)
			}
		}
		q.grounded = grounded
	}
}
