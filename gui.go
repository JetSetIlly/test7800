package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type gui struct {
	endGui chan bool
}

func (g *gui) Update() error {
	select {
	case <-g.endGui:
		return ebiten.Termination
	default:
	}
	return nil
}

func (g *gui) Draw(screen *ebiten.Image) {
}

func (g *gui) Layout(width, height int) (int, int) {
	return width, height
}

func startGui(endGui chan bool) error {
	return ebiten.RunGame(&gui{
		endGui: endGui,
	})
}
