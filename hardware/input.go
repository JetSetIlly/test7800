package hardware

import (
	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware/peripherals"
	"github.com/jetsetilly/test7800/logger"
)

type peripheral interface {
	IsAnalogue() bool
	IsController() bool
	Reset()
	Unplug()
	Update(inp gui.Input) error
	Tick()
}

type filteringPeripheral interface {
	SetInputFilter(inputSource gui.InputSource, secondary bool)
}

func (con *Console) handleInput() {
	var drained bool
	for !drained {
		select {
		default:
			drained = true
		case inp := <-con.g.UserInput:
			if inp.Action == gui.AnalogueSelect && inp.Data.(bool) {
				switch inp.Port {
				case gui.Players:
					// on-demand analogue select limited to the right player. this works nicely for
					// some games but the best way of enabling paddles is to have the paddles set in
					// the a78 head
					if con.players[1].IsController() && !con.players[1].IsAnalogue() {
						if _, ok := con.players[1].(*peripherals.Paddles); !ok {
							logger.Log(logger.Allow, "controllers", "plugging paddle into right player port")
							con.players[1].Unplug()
							con.players[1] = peripherals.NewPaddles(con.RIOT, con.TIA, true)
							con.players[1].Reset()
						}
					}
				}
			} else {
				switch inp.Port {
				case gui.Panel:
					con.panel.Update(inp)
				case gui.Players:
					con.players[0].Update(inp)
					con.players[1].Update(inp)
				}
			}
		}
	}
}

func (con *Console) SetPlayers(inputSources []gui.InputSource) error {
	if c, ok := con.players[0].(filteringPeripheral); ok {
		if len(inputSources) > 0 {
			c.SetInputFilter(inputSources[0], true)
		}
		if len(inputSources) > 2 {
			c.SetInputFilter(inputSources[2], false)
		}
	}

	if c, ok := con.players[1].(filteringPeripheral); ok {
		if len(inputSources) > 1 {
			c.SetInputFilter(inputSources[1], true)
		}
		if len(inputSources) > 3 {
			c.SetInputFilter(inputSources[3], false)
		}
	}

	con.inputSources = inputSources[:]

	return nil
}
