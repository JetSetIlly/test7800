package peripherals

import (
	"fmt"

	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware/riot"
	"github.com/jetsetilly/test7800/hardware/tia"
)

type paddle struct {
	tia       PaddlesTIA
	inptx     tia.Register
	swchaMask uint8

	// values indicating paddle state
	charge     uint8
	resistance int
	ticks      int
}

func (pdl *paddle) ground() {
	pdl.charge = 0x00
	pdl.ticks = 0
	pdl.tia.PortWrite(pdl.inptx, pdl.charge, 0x00)
}

func (pdl *paddle) changeResistance(v int) {
	pdl.resistance -= v
	pdl.resistance = max(pdl.resistance, 0)
	pdl.resistance = min(pdl.resistance, 255)
}

func (pdl *paddle) tick() {
	if pdl.charge < 0xff {
		pdl.ticks++
		if pdl.ticks >= pdl.resistance {
			pdl.ticks = 0
			pdl.charge++
			pdl.charge = min(pdl.charge, 255)
			pdl.tia.PortWrite(pdl.inptx, pdl.charge, 0x00)
		}
	}
}

// Paddles represent a pair of paddles that plug into a single player port
type Paddles struct {
	portRight bool
	riot      RIOT
	tia       PaddlesTIA
	paddles   [2]paddle
	grounded  bool
}

func NewPaddles(r RIOT, t PaddlesTIA, portRight bool) *Paddles {
	pdl := &Paddles{
		portRight: portRight,
		riot:      r,
		tia:       t,
	}

	pdl.paddles[0].tia = t
	pdl.paddles[1].tia = t

	if portRight {
		pdl.paddles[0].inptx = tia.INPT2
		pdl.paddles[1].inptx = tia.INPT3
		pdl.paddles[0].swchaMask = 0x08
		pdl.paddles[1].swchaMask = 0x04
	} else {
		pdl.paddles[0].inptx = tia.INPT0
		pdl.paddles[1].inptx = tia.INPT1
		pdl.paddles[0].swchaMask = 0x80
		pdl.paddles[1].swchaMask = 0x40
	}

	return pdl
}

func (pdl *Paddles) IsAnalogue() bool {
	return true
}

func (pdl *Paddles) Reset() {
	pdl.tia.PortWrite(pdl.paddles[0].inptx, 0x00, 0xf0)
	pdl.tia.PortWrite(pdl.paddles[1].inptx, 0x00, 0xf0)
	pdl.riot.PortWrite(riot.SWCHA, pdl.paddles[0].swchaMask, ^pdl.paddles[0].swchaMask)
	pdl.riot.PortWrite(riot.SWCHA, pdl.paddles[0].swchaMask, ^pdl.paddles[1].swchaMask)
}

func (pdl *Paddles) Unplug() {
	pdl.tia.PortWrite(pdl.paddles[0].inptx, 0x00, 0xf0)
	pdl.tia.PortWrite(pdl.paddles[1].inptx, 0x00, 0xf0)
	pdl.riot.PortWrite(riot.SWCHA, 0x0c, 0xf3)
}

func (pdl *Paddles) Update(inp gui.Input) error {
	switch inp.Action {
	case gui.PaddleFire:
		d, ok := inp.Data.(gui.PaddleFireData)
		if !ok {
			return fmt.Errorf("paddle: illegal paddle fire data")
		}
		if d.Fire {
			switch d.Paddle {
			case 0:
				pdl.riot.PortWrite(riot.SWCHA, 0x00, ^pdl.paddles[0].swchaMask)
			case 1:
				pdl.riot.PortWrite(riot.SWCHA, 0x00, ^pdl.paddles[1].swchaMask)
			default:
				return fmt.Errorf("paddle: illegal paddle fire data: no such paddle: %d", d.Paddle)
			}
		} else {
			switch d.Paddle {
			case 0:
				pdl.riot.PortWrite(riot.SWCHA, pdl.paddles[0].swchaMask, ^pdl.paddles[0].swchaMask)
			case 1:
				pdl.riot.PortWrite(riot.SWCHA, pdl.paddles[1].swchaMask, ^pdl.paddles[1].swchaMask)
			default:
				return fmt.Errorf("paddle: illegal paddle fire data: no such paddle: %d", d.Paddle)
			}
		}
	case gui.PaddleMove:
		d, ok := inp.Data.(gui.PaddleMoveData)
		if !ok {
			return fmt.Errorf("paddle: illegal paddle move data")
		}

		switch d.Paddle {
		case 0:
			pdl.paddles[0].changeResistance(d.Delta)
		case 1:
			pdl.paddles[1].changeResistance(d.Delta)
		default:
			return fmt.Errorf("paddle: illegal paddle fire data: no such paddle: %d", d.Paddle)
		}
	}

	return nil
}

func (pdl *Paddles) Tick() {
	if !pdl.grounded && pdl.tia.PaddlesGrounded() {
		pdl.grounded = true
		pdl.paddles[0].ground()
		pdl.paddles[1].ground()
	} else {
		pdl.grounded = false
		pdl.paddles[0].tick()
		pdl.paddles[1].tick()
	}
}
