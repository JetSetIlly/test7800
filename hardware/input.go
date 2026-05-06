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
					if con.players[0].IsController() && !con.players[0].IsAnalogue() {
						if _, ok := con.players[0].(*peripherals.Paddles); !ok {
							logger.Log(logger.Allow, "controllers", "plugging paddle into player 0 port")
							con.players[0].Unplug()
							con.players[0] = peripherals.NewPaddles(con.RIOT, con.TIA, false)
							con.players[0].Reset()
						}
					}
					if con.players[1].IsController() && !con.players[1].IsAnalogue() {
						if _, ok := con.players[1].(*peripherals.Paddles); !ok {
							logger.Log(logger.Allow, "controllers", "plugging paddle into player 1 port")
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
	if len(inputSources) == 0 {
		return nil
	}

	if len(inputSources) < 4 {
		panic("console.SetPlayers() should be called with a slice with at least four elements")
	}

	if c, ok := con.players[0].(filteringPeripheral); ok {
		c.SetInputFilter(inputSources[0], true)
		c.SetInputFilter(inputSources[2], false)
	}

	if c, ok := con.players[1].(filteringPeripheral); ok {
		c.SetInputFilter(inputSources[1], true)
		c.SetInputFilter(inputSources[3], false)
	}

	con.inputSources = inputSources[:]

	return nil
}
