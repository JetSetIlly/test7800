package gui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/jetsetilly/test7800/io"
	input "github.com/quasilyte/ebitengine-input"
)

type gui struct {
	started bool

	endGui    chan bool
	rendering chan *image.RGBA
	inp       chan io.Input

	image  *ebiten.Image
	width  int
	height int

	inputHandler *input.Handler
	inputSystem  input.System
}

const (
	ActionStickLeft    = input.Action(io.StickLeft)
	ActionStickUp      = input.Action(io.StickUp)
	ActionStickRight   = input.Action(io.StickRight)
	ActionStickDown    = input.Action(io.StickDown)
	ActionStickButtonA = input.Action(io.StickButtonA)
)

func (g *gui) initialise() {
	keymap := input.Keymap{
		ActionStickLeft:    {input.KeyGamepadLeft, input.KeyLeft},
		ActionStickUp:      {input.KeyGamepadUp, input.KeyUp},
		ActionStickRight:   {input.KeyGamepadRight, input.KeyRight},
		ActionStickDown:    {input.KeyGamepadDown, input.KeyDown},
		ActionStickButtonA: {input.KeyGamepadA, input.KeySpace, input.KeyX},
	}
	g.inputHandler = g.inputSystem.NewHandler(uint8(0), keymap)
	g.started = true
}

func (g *gui) input() {
	g.inputSystem.Update()

	var inp io.Input

	if g.inputHandler.ActionIsJustPressed(ActionStickLeft) {
		inp = io.Input{Action: io.StickLeft}
	}
	if g.inputHandler.ActionIsJustPressed(ActionStickUp) {
		inp = io.Input{Action: io.StickUp}
	}
	if g.inputHandler.ActionIsJustPressed(ActionStickRight) {
		inp = io.Input{Action: io.StickRight}
	}
	if g.inputHandler.ActionIsJustPressed(ActionStickDown) {
		inp = io.Input{Action: io.StickDown}
	}
	if g.inputHandler.ActionIsJustPressed(ActionStickButtonA) {
		inp = io.Input{Action: io.StickButtonA}
	}

	if g.inputHandler.ActionIsJustReleased(ActionStickLeft) {
		inp = io.Input{Action: io.StickLeft, Release: true}
	}
	if g.inputHandler.ActionIsJustReleased(ActionStickUp) {
		inp = io.Input{Action: io.StickUp, Release: true}
	}
	if g.inputHandler.ActionIsJustReleased(ActionStickRight) {
		inp = io.Input{Action: io.StickRight, Release: true}
	}
	if g.inputHandler.ActionIsJustReleased(ActionStickDown) {
		inp = io.Input{Action: io.StickDown, Release: true}
	}
	if g.inputHandler.ActionIsJustReleased(ActionStickButtonA) {
		inp = io.Input{Action: io.StickButtonA, Release: true}
	}

	if inp.Action != io.Nothing {
		select {
		case g.inp <- inp:
		default:
		}
	}
}

func (g *gui) Update() error {
	if !g.started {
		g.initialise()
	}

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

func Launch(endGui chan bool, rendering chan *image.RGBA, inp chan io.Input) error {
	ebiten.SetWindowTitle("test7800")
	ebiten.SetVsyncEnabled(true)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowPosition(10, 10)
	ebiten.SetTPS(ebiten.SyncWithFPS)

	g := &gui{
		endGui:    endGui,
		rendering: rendering,
		inp:       inp,
	}

	g.inputSystem.Init(input.SystemConfig{
		DevicesEnabled: input.AnyDevice,
	})

	return ebiten.RunGame(g)
}
