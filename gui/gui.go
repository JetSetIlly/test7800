package gui

import (
	"fmt"
	"image/color"
	"math"

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

	state ui.State

	main    *ebiten.Image
	overlay *ebiten.Image
	prev    *ebiten.Image
	prevID  int
	cursor  [2]int

	width  int
	height int

	// a simple counter used to implement a fade-in/fade-out effect for the
	// debugging cursor
	cursorFrame int
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
		case ebiten.KeyF1:
			inp = ui.Input{Action: ui.Select, Release: true}
		case ebiten.KeyF2:
			inp = ui.Input{Action: ui.Reset, Release: true}
		case ebiten.KeyF4:
			inp = ui.Input{Action: ui.P0Pro, Release: true}
		case ebiten.KeyF5:
			inp = ui.Input{Action: ui.P1Pro, Release: true}
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
		case ebiten.KeyF1:
			inp = ui.Input{Action: ui.Select}
		case ebiten.KeyF2:
			inp = ui.Input{Action: ui.Reset}
		case ebiten.KeyF4:
			inp = ui.Input{Action: ui.P0Pro}
		case ebiten.KeyF5:
			inp = ui.Input{Action: ui.P1Pro}
		}

		select {
		case g.ui.UserInput <- inp:
		default:
			return
		}
	}
}

func (g *gui) Update() error {
	// deal with quit condition
	select {
	case <-g.endGui:
		return ebiten.Termination
	default:
	}

	// handle user input
	g.input()

	// change state if necessary
	select {
	case g.state = <-g.ui.State:
	default:
	}

	// run option update function
	if g.ui.UpdateGUI != nil {
		err := g.ui.UpdateGUI()
		if err != nil {
			return fmt.Errorf("ebiten: %w", err)
		}
	}

	// retrieve any pending images
	select {
	case img := <-g.ui.SetImage:
		g.cursor = img.Cursor

		dim := img.Main.Bounds()
		if g.main == nil || (g.main == nil && g.main.Bounds() != dim) {
			g.width = dim.Dx()
			g.height = dim.Dy()
			g.main = ebiten.NewImage(g.width, g.height)
			g.prev = ebiten.NewImage(g.width, g.height)
			g.overlay = ebiten.NewImage(g.width, g.height)
		}

		g.main.WritePixels(img.Main.Pix)

		if img.Prev != nil && img.PrevID != g.prevID {
			g.prevID = img.PrevID
			g.prev.WritePixels(img.Prev.Pix)
		}

		if img.Overlay != nil {
			g.overlay.WritePixels(img.Overlay.Pix)
		}

	default:
	}

	return nil
}

func (g *gui) Draw(screen *ebiten.Image) {
	g.cursorFrame++

	if g.main != nil {
		if g.prev != nil {
			var op ebiten.DrawImageOptions
			op.ColorScale.SetR(0.2)
			op.ColorScale.SetG(0.2)
			op.ColorScale.SetB(0.2)
			op.ColorScale.SetA(1.0)
			screen.DrawImage(g.prev, &op)
		}
		if g.main != nil {
			var op ebiten.DrawImageOptions
			op.Blend = ebiten.BlendSourceOver
			screen.DrawImage(g.main, &op)
		}
		if g.overlay != nil {
			var op ebiten.DrawImageOptions
			op.Blend = ebiten.BlendLighter
			screen.DrawImage(g.overlay, &op)
		}

		// draw cursor if emulation is paused
		if g.state == ui.StatePaused {
			v := uint8((math.Sin(float64(g.cursorFrame/10))*0.5 + 0.5) * 255)
			screen.Set(g.cursor[0], g.cursor[1], color.RGBA{R: v, G: v, B: v, A: 255})
			screen.Set(g.cursor[0]+1, g.cursor[1], color.RGBA{R: v, G: v, B: v, A: 255})
			screen.Set(g.cursor[0], g.cursor[1]+1, color.RGBA{R: v, G: v, B: v, A: 255})
			screen.Set(g.cursor[0]+1, g.cursor[1]+1, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
}

func (g *gui) Layout(width, height int) (int, int) {
	if g.main != nil {
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
