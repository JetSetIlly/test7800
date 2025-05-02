package gui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/jetsetilly/test7800/ui"

	"github.com/hajimehoshi/ebiten/v2/audio"

	tia "github.com/jetsetilly/test7800/hardware/tia/audio"
)

type gui struct {
	started bool

	endGui chan bool
	ui     *ui.UI

	image  *ebiten.Image
	width  int
	height int
}

func (g *gui) input() {
	var pressed []ebiten.Key
	var released []ebiten.Key
	pressed = inpututil.AppendJustPressedKeys(pressed)
	released = inpututil.AppendJustReleasedKeys(released)

	var inp ui.Input

	for _, r := range released {
		switch r {
		case ebiten.KeyArrowLeft:
			inp = ui.Input{Action: ui.StickLeft, Release: true}
		case ebiten.KeyArrowRight:
			inp = ui.Input{Action: ui.StickRight, Release: true}
		case ebiten.KeyArrowUp:
			inp = ui.Input{Action: ui.StickUp, Release: true}
		case ebiten.KeyArrowDown:
			inp = ui.Input{Action: ui.StickDown, Release: true}
		case ebiten.KeySpace:
			inp = ui.Input{Action: ui.StickButtonA, Release: true}
		}

		select {
		case g.ui.UserInput <- inp:
		default:
			return
		}
	}

	for _, p := range pressed {
		switch p {
		case ebiten.KeyArrowLeft:
			inp = ui.Input{Action: ui.StickLeft}
		case ebiten.KeyArrowRight:
			inp = ui.Input{Action: ui.StickRight}
		case ebiten.KeyArrowUp:
			inp = ui.Input{Action: ui.StickUp}
		case ebiten.KeyArrowDown:
			inp = ui.Input{Action: ui.StickDown}
		case ebiten.KeySpace:
			inp = ui.Input{Action: ui.StickButtonA}
		}

		select {
		case g.ui.UserInput <- inp:
		default:
			return
		}
	}
}

func (g *gui) Update() error {
	g.input()

	if g.ui.UpdateGUI != nil {
		err := g.ui.UpdateGUI()
		if err != nil {
			return fmt.Errorf("ebiten: %w", err)
		}
	}

	select {
	case <-g.endGui:
		return ebiten.Termination
	case img := <-g.ui.SetImage:
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

func (g *gui) Draw(screen *ebiten.Image) {
	if g.image != nil {
		op := &ebiten.DrawImageOptions{}
		screen.DrawImage(g.image, op)
	}
}

func (g *gui) Layout(width, height int) (int, int) {
	if g.image != nil {
		return g.width, g.height
	}
	return width, height
}

func Launch(endGui chan bool, ui *ui.UI) error {
	ebiten.SetWindowTitle("test7800")
	ebiten.SetVsyncEnabled(true)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowPosition(10, 10)
	ebiten.SetTPS(ebiten.SyncWithFPS)

	g := &gui{
		endGui: endGui,
		ui:     ui,
	}

	if ui.RegisterAudio != nil {
		audioctx := audio.NewContext(tia.AverageSampleFreq)
		p, err := audioctx.NewPlayer(<-ui.RegisterAudio)
		if err != nil {
			return err
		}
		p.Play()
	}

	return ebiten.RunGame(g)
}
