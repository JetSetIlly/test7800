package peripherals

import (
	"fmt"

	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware/riot"
	"github.com/jetsetilly/test7800/hardware/tia"
)

type peripheral interface {
	IsAnalogue() bool
	IsController() bool
	Reset()
	Unplug()
	Update(inp gui.Input) error
	Tick()
	SWCHA() (uint8, uint8)
	Button() (tia.Register, bool)
}

type filteringPeripheral interface {
	SetInputFilter(inputSource gui.InputSource, secondary bool)
}

type Quadtari struct {
	riot      RIOT
	tia       GroundedTIA
	portRight bool

	First  peripheral
	Second peripheral

	// whether the TIA returned grounded on the most recent tick
	grounded    bool
	groundDelay int
}

func NewQuadtari(r RIOT, tia GroundedTIA, portRight bool) *Quadtari {
	return &Quadtari{
		riot:      r,
		tia:       tia,
		portRight: portRight,
	}
}

func (q *Quadtari) IsAnalogue() bool {
	if q.tia.Grounded() {
		return q.First.IsAnalogue()
	}
	return q.Second.IsAnalogue()
}

func (q *Quadtari) IsController() bool {
	if q.tia.Grounded() {
		return q.First.IsController()
	}
	return q.Second.IsController()
}

func (q *Quadtari) Reset() {
	q.First.Reset()
	q.Second.Reset()
	if q.portRight {
		q.tia.PortWrite(tia.INPT2, 0x00, 0x7f)
		q.tia.PortWrite(tia.INPT3, 0x80, 0x7f)
	} else {
		q.tia.PortWrite(tia.INPT0, 0x00, 0x7f)
		q.tia.PortWrite(tia.INPT1, 0x80, 0x7f)
	}
}

func (q *Quadtari) Unplug() {
	q.First.Unplug()
	q.Second.Unplug()
	if q.portRight {
		q.tia.PortWrite(tia.INPT2, 0x00, 0x7f)
		q.tia.PortWrite(tia.INPT3, 0x00, 0x7f)
	} else {
		q.tia.PortWrite(tia.INPT0, 0x00, 0x7f)
		q.tia.PortWrite(tia.INPT1, 0x00, 0x7f)
	}
}

func (q *Quadtari) Update(inp gui.Input) error {
	err := q.First.Update(inp)
	if err != nil {
		return fmt.Errorf("quadtari: %w", err)
	}
	err = q.Second.Update(inp)
	if err != nil {
		return fmt.Errorf("quadtari: %w", err)
	}
	return nil
}

func (q *Quadtari) Tick() {
	q.First.Tick()
	q.Second.Tick()

	grounded := q.tia.Grounded()
	if grounded != q.grounded {
		if q.grounded {
			data, mask := q.First.SWCHA()
			q.riot.PortWrite(riot.SWCHA, data, mask)
			if reg, fire := q.First.Button(); fire {
				q.tia.PortWrite(reg, 0x00, 0x7f)
			} else {
				q.tia.PortWrite(reg, 0x80, 0x7f)
			}
		} else {
			data, mask := q.Second.SWCHA()
			q.riot.PortWrite(riot.SWCHA, data, mask)
			if reg, fire := q.Second.Button(); fire {
				q.tia.PortWrite(reg, 0x00, 0x7f)
			} else {
				q.tia.PortWrite(reg, 0x80, 0x7f)
			}
		}

		q.grounded = grounded
	}
}

func (q *Quadtari) SetInputFilter(inputSource gui.InputSource, secondary bool) {
	if secondary {
		if c, ok := q.Second.(filteringPeripheral); ok {
			c.SetInputFilter(inputSource, secondary)
		}
	} else {
		if c, ok := q.First.(filteringPeripheral); ok {
			c.SetInputFilter(inputSource, secondary)
		}
	}
}
