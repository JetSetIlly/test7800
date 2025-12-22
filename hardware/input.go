package hardware

import (
	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware/peripherals"
	"github.com/jetsetilly/test7800/logger"
)

func (con *Console) handleInput() {
	var drained bool
	for !drained {
		select {
		default:
			drained = true
		case inp := <-con.g.UserInput:
			if inp.Action == gui.PaddleSelect && inp.Data.(bool) {
				switch inp.Port {
				case gui.Player0:
					if _, ok := con.players[0].(*peripherals.Paddles); !ok {
						logger.Log(logger.Allow, "controllers", "plugging paddle into player 0 port")
						con.players[0].Unplug()
						con.players[0] = peripherals.NewPaddles(con.RIOT, con.TIA, false)
						con.players[0].Reset()
					}
				case gui.Undefined:
					fallthrough
				case gui.Player1:
					logger.Log(logger.Allow, "controllers", "plugging paddle into player 1 port")
					if _, ok := con.players[1].(*peripherals.Paddles); !ok {
						con.players[1].Unplug()
						con.players[1] = peripherals.NewPaddles(con.RIOT, con.TIA, true)
						con.players[1].Reset()
					}
				}
			} else {
				switch inp.Port {
				case gui.Panel:
					con.panel.Update(inp)
				case gui.Player0:
					con.players[0].Update(inp)
				case gui.Player1:
					con.players[1].Update(inp)
				case gui.Undefined:
					con.players[0].Update(inp)
					con.players[1].Update(inp)
				}
			}
		}
	}
}
