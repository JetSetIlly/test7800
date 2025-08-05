package ebiten

import (
	"fmt"
	"image/color"
	"math"
	"sync"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/logger"
	"github.com/jetsetilly/test7800/version"
)

type audioPlayer struct {
	p *oto.Player
	r gui.AudioReader

	// the state field is accessed by the Read() function via the audio
	// engine, and by the GUI which is in another goroutine. access to the state
	// field therefore, is proctected by a mutex
	crit  sync.Mutex
	state gui.State
}

func (a *audioPlayer) setState(state gui.State) {
	a.crit.Lock()
	defer a.crit.Unlock()
	a.state = state
}

func (a *audioPlayer) Read(buf []uint8) (int, error) {
	a.crit.Lock()
	defer a.crit.Unlock()
	if a.state != gui.StateRunning {
		return 0, nil
	}

	const prefetch = 2048

	sz := a.p.BufferedSize()
	if sz < prefetch {
		a.r.Prefetch(prefetch - sz)
	}

	n, err := a.r.Read(buf)
	if err != nil {
		return 0, err
	}
	return n, nil
}

type guiEbiten struct {
	g *gui.GUI

	started bool
	endGui  chan bool

	state gui.State

	main    *ebiten.Image
	overlay *ebiten.Image
	prev    *ebiten.Image
	prevID  int
	cursor  [2]int

	// width/height of incoming image from emulation. not to be confused with window dimensions
	width  int
	height int

	// a simple counter used to implement a fade-in/fade-out effect for the
	// debugging cursor
	cursorFrame int

	// the audio player can be stopped and recreated as required
	audio audioPlayer

	// the hardware of the difficulty switches have an implicit state (because
	// they are switches) that we can't effectively store any other way besides
	// keeping track of the physical state.
	proDifficulty [2]bool
}

func (eg *guiEbiten) input() {
	var pressed []ebiten.Key
	var released []ebiten.Key
	pressed = inpututil.AppendJustPressedKeys(pressed)
	released = inpututil.AppendJustReleasedKeys(released)

	var inp gui.Input

	for _, p := range released {
		switch p {
		case ebiten.KeyArrowLeft:
			inp = gui.Input{Action: gui.StickLeft}
		case ebiten.KeyArrowRight:
			inp = gui.Input{Action: gui.StickRight}
		case ebiten.KeyArrowUp:
			inp = gui.Input{Action: gui.StickUp}
		case ebiten.KeyArrowDown:
			inp = gui.Input{Action: gui.StickDown}
		case ebiten.KeySpace:
			inp = gui.Input{Action: gui.StickButtonA}
		case ebiten.KeyB:
			inp = gui.Input{Action: gui.StickButtonB}
		case ebiten.KeyF1:
			inp = gui.Input{Action: gui.Select}
		case ebiten.KeyF2:
			inp = gui.Input{Action: gui.Start}
		case ebiten.KeyF3:
			inp = gui.Input{Action: gui.Pause}
		case ebiten.KeyF4:
			inp = gui.Input{Action: gui.P0Pro, Set: eg.proDifficulty[0]}
		case ebiten.KeyF5:
			inp = gui.Input{Action: gui.P1Pro, Set: eg.proDifficulty[1]}
		}

		select {
		case eg.g.UserInput <- inp:
		default:
			return
		}
	}

	for _, r := range pressed {
		switch r {
		case ebiten.KeyArrowLeft:
			inp = gui.Input{Action: gui.StickLeft, Set: true}
		case ebiten.KeyArrowRight:
			inp = gui.Input{Action: gui.StickRight, Set: true}
		case ebiten.KeyArrowUp:
			inp = gui.Input{Action: gui.StickUp, Set: true}
		case ebiten.KeyArrowDown:
			inp = gui.Input{Action: gui.StickDown, Set: true}
		case ebiten.KeySpace:
			inp = gui.Input{Action: gui.StickButtonA, Set: true}
		case ebiten.KeyB:
			inp = gui.Input{Action: gui.StickButtonB, Set: true}
		case ebiten.KeyF1:
			inp = gui.Input{Action: gui.Select, Set: true}
		case ebiten.KeyF2:
			inp = gui.Input{Action: gui.Start, Set: true}
		case ebiten.KeyF3:
			inp = gui.Input{Action: gui.Pause, Set: true}
		case ebiten.KeyF4:
			eg.proDifficulty[0] = !eg.proDifficulty[0]
		case ebiten.KeyF5:
			eg.proDifficulty[1] = !eg.proDifficulty[1]
		}

		select {
		case eg.g.UserInput <- inp:
		default:
			return
		}
	}
}

func (eg *guiEbiten) Update() error {
	// deal with quit condition
	select {
	case <-eg.endGui:
		if eg.audio.p != nil {
			eg.audio.p.Close()
		}
		return ebiten.Termination
	default:
	}

	// handle user input
	eg.input()

	// change state if necessary
	select {
	case eg.state = <-eg.g.State:
		eg.audio.setState(eg.state)
	default:
	}

	// create audio if necessary
	if eg.g.AudioSetup != nil {
		select {
		case s := <-eg.g.AudioSetup:
			if eg.g.AudioSetup != nil {
				if eg.audio.p != nil {
					err := eg.audio.p.Close()
					if err != nil {
						return fmt.Errorf("ebiten: %w", err)
					}
				}

				ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
					SampleRate:   int(s.Freq),
					ChannelCount: 2,
					Format:       oto.FormatSignedInt16LE,
				})
				if err != nil {
					return fmt.Errorf("ebiten: %w", err)
				}

				<-ready

				eg.audio.r = s.Read
				eg.audio.p = ctx.NewPlayer(&eg.audio)
				eg.audio.p.Play()
			}

		default:
		}
	}

	// run option update function
	if eg.g.UpdateGUI != nil {
		err := eg.g.UpdateGUI()
		if err != nil {
			return fmt.Errorf("ebiten: %w", err)
		}
	}

	// retrieve any pending images
	select {
	case img := <-eg.g.SetImage:
		eg.cursor = img.Cursor

		dim := img.Main.Bounds()
		if eg.main == nil || (eg.main == nil && eg.main.Bounds() != dim) {
			eg.width = dim.Dx()
			eg.height = dim.Dy()
			eg.main = ebiten.NewImage(eg.width, eg.height)
			eg.prev = ebiten.NewImage(eg.width, eg.height)
			eg.overlay = ebiten.NewImage(eg.width, eg.height)
		}

		eg.main.WritePixels(img.Main.Pix)

		if img.Prev != nil && img.PrevID != eg.prevID {
			eg.prevID = img.PrevID
			eg.prev.WritePixels(img.Prev.Pix)
		}

		if img.Overlay != nil {
			eg.overlay.WritePixels(img.Overlay.Pix)
		}

	default:
	}

	return nil
}

func (eg *guiEbiten) Draw(screen *ebiten.Image) {
	eg.cursorFrame++

	if eg.main != nil {
		if eg.prev != nil {
			var op ebiten.DrawImageOptions
			op.ColorScale.SetR(0.2)
			op.ColorScale.SetG(0.2)
			op.ColorScale.SetB(0.2)
			op.ColorScale.SetA(1.0)
			screen.DrawImage(eg.prev, &op)
		}
		if eg.main != nil {
			var op ebiten.DrawImageOptions
			op.Blend = ebiten.BlendSourceOver
			screen.DrawImage(eg.main, &op)
		}
		if eg.overlay != nil {
			var op ebiten.DrawImageOptions
			op.Blend = ebiten.BlendLighter
			screen.DrawImage(eg.overlay, &op)
		}

		// draw cursor if emulation is paused
		if eg.state == gui.StatePaused {
			v := uint8((math.Sin(float64(eg.cursorFrame/10))*0.5 + 0.5) * 255)
			screen.Set(eg.cursor[0], eg.cursor[1], color.RGBA{R: v, G: v, B: v, A: 255})
			screen.Set(eg.cursor[0]+1, eg.cursor[1], color.RGBA{R: v, G: v, B: v, A: 255})
			screen.Set(eg.cursor[0], eg.cursor[1]+1, color.RGBA{R: v, G: v, B: v, A: 255})
			screen.Set(eg.cursor[0]+1, eg.cursor[1]+1, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}

	// if window is being close
	if ebiten.IsWindowBeingClosed() {
		err := onCloseWindow()
		if err != nil {
			logger.Log(logger.Allow, "gui", err.Error())
			return
		}
	}
}

func (eg *guiEbiten) Layout(width, height int) (int, int) {
	if eg.main != nil {
		return eg.width, eg.height
	}
	return width, height
}

func Launch(endGui chan bool, g *gui.GUI) error {
	ebiten.SetWindowTitle(version.Title())
	ebiten.SetVsyncEnabled(true)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowPosition(10, 10)
	ebiten.SetTPS(ebiten.SyncWithFPS)

	eg := &guiEbiten{
		endGui: endGui,
		g:      g,
		state:  gui.StateRunning,
		audio: audioPlayer{
			state: gui.StateRunning,
		},
	}

	err := onWindowOpen()
	if err != nil {
		logger.Log(logger.Allow, "gui", err.Error())
	}

	return ebiten.RunGame(eg)
}
