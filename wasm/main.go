package main

import (
	"github.com/jetsetilly/test7800/debugger/dbg"
	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware"
	"github.com/jetsetilly/test7800/ui"
)

func main() {
	ui := ui.NewUI()
	ctx, err := dbg.Create("7800", "PAL")
	if err != nil {
		panic(err)
	}
	con := hardware.Create(&ctx, ui)
	ui.UpdateGUI = func() error {
		fn := con.MARIA.Coords.Frame
		for con.MARIA.Coords.Frame == fn {
			err := con.Step()
			if err != nil {
				return err
			}
		}
		return nil
	}
	gui.Launch(nil, ui, false)
}
