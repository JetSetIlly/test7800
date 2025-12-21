package peripherals

import (
	"fmt"

	"github.com/jetsetilly/test7800/gui"
)

type paddle struct {
	inptx uint8

	// button data is always written to SWCHA but which bit depends on the paddle
	buttonMask uint8

	// values indicating paddle state
	charge     uint8
	resistance int
	ticks      int

	// the state of the fire button
	fire bool
}

// Paddles represent a pair of paddles that plug into a single player port
type Paddles struct {
	portRight bool
	riot      RIOT
	tia       TIA
	paddles   [2]paddle
}

func NewPaddles(riot RIOT, tia TIA, portRight bool) *Paddles {
	pd := &Paddles{
		portRight: portRight,
		riot:      riot,
		tia:       tia,
	}

	if portRight {
		pd.paddles[0].inptx = 2
		pd.paddles[1].inptx = 3
	} else {
		pd.paddles[0].inptx = 0
		pd.paddles[1].inptx = 1
	}

	return pd
}

func (pd *Paddles) Reset() {
}

func (pd *Paddles) Update(inp gui.Input) error {
	switch inp.Action {
	case gui.PaddleFire:
		fmt.Println("fire")
	case gui.PaddleSet:
		fmt.Println("set")
	}
	return nil
}

func (pd *Paddles) Step() {
}
