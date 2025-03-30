package gui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type gui struct {
	endGui    chan bool
	rendering chan *image.RGBA
	frame     *ebiten.Image
	width     int
	height    int
}

func (g *gui) Update() error {
	select {
	case <-g.endGui:
		return ebiten.Termination
	case img := <-g.rendering:
		dim := img.Bounds()
		if g.frame == nil || (g.frame == nil && g.frame.Bounds() != dim) {
			g.width = dim.Dx()
			g.height = dim.Dy()
			g.frame = ebiten.NewImage(g.width, g.height)
		}
		g.frame.WritePixels(img.Pix)
	default:
	}
	return nil
}

const (
	pixelWidth = 2
)

func (g *gui) Draw(screen *ebiten.Image) {
	if g.frame != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(pixelWidth, 1)
		screen.DrawImage(g.frame, op)
	}
}

func (g *gui) Layout(width, height int) (int, int) {
	if g.frame != nil {
		return g.width * pixelWidth, g.height
	}
	return width, height
}

func Launch(endGui chan bool, rendering chan *image.RGBA) error {
	ebiten.SetWindowTitle("test7800")
	ebiten.SetVsyncEnabled(true)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowPosition(10, 10)
	ebiten.SetTPS(ebiten.SyncWithFPS)

	return ebiten.RunGame(&gui{
		endGui:    endGui,
		rendering: rendering,
	})
}
