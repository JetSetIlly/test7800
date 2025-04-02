package gui

import (
	"image"
	"io"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/jetsetilly/test7800/ui"
)

type gui struct {
	started bool

	endGui    chan bool
	rendering chan *image.RGBA
	inp       chan ui.Input
	sound     *sound

	image  *ebiten.Image
	width  int
	height int
}

func (g *gui) input() {
	var pressed []ebiten.Key
	var released []ebiten.Key
	pressed = inpututil.AppendJustPressedKeys(pressed)
	released = inpututil.AppendJustReleasedKeys(released)

	for _, p := range pressed {
		switch p {
		case ebiten.KeyArrowLeft:
			g.inp <- ui.Input{Action: ui.StickLeft}
		case ebiten.KeyArrowRight:
			g.inp <- ui.Input{Action: ui.StickRight}
		case ebiten.KeyArrowUp:
			g.inp <- ui.Input{Action: ui.StickUp}
		case ebiten.KeyArrowDown:
			g.inp <- ui.Input{Action: ui.StickDown}
		case ebiten.KeySpace:
			g.inp <- ui.Input{Action: ui.StickButtonA}
		}
	}

	for _, r := range released {
		switch r {
		case ebiten.KeyArrowLeft:
			g.inp <- ui.Input{Action: ui.StickLeft, Release: true}
		case ebiten.KeyArrowRight:
			g.inp <- ui.Input{Action: ui.StickRight, Release: true}
		case ebiten.KeyArrowUp:
			g.inp <- ui.Input{Action: ui.StickUp, Release: true}
		case ebiten.KeyArrowDown:
			g.inp <- ui.Input{Action: ui.StickDown, Release: true}
		case ebiten.KeySpace:
			g.inp <- ui.Input{Action: ui.StickButtonA, Release: true}
		}
	}
}

func (g *gui) Update() error {
	g.input()

	select {
	case <-g.endGui:
		return ebiten.Termination
	case img := <-g.rendering:
		dim := img.Bounds()
		if g.image == nil || (g.image == nil && g.image.Bounds() != dim) {
			g.width = dim.Dx()
			g.height = dim.Dy()
			g.image = ebiten.NewImage(g.width, g.height)
		}
		g.image.WritePixels(img.Pix)

	default:
	}

	return nil
}

const (
	pixelWidth = 2
)

func (g *gui) Draw(screen *ebiten.Image) {
	if g.image != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(pixelWidth, 1)
		screen.DrawImage(g.image, op)
	}
}

func (g *gui) Layout(width, height int) (int, int) {
	if g.image != nil {
		return g.width * pixelWidth, g.height
	}
	return width, height
}

func Launch(endGui chan bool, rendering chan *image.RGBA, snd chan io.Reader, inp chan ui.Input) error {
	ebiten.SetWindowTitle("test7800")
	ebiten.SetVsyncEnabled(true)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowPosition(10, 10)
	ebiten.SetTPS(ebiten.SyncWithFPS)

	g := &gui{
		endGui:    endGui,
		rendering: rendering,
		inp:       inp,
		sound:     createAudio(snd),
	}

	return ebiten.RunGame(g)
}
