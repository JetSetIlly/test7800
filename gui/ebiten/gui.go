package ebiten

import (
	"fmt"
	"image/color"
	"math"
	"sync"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/ebiten/v2"
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
		a.r.Nudge()
	}

	n, err := a.r.Read(buf)
	if err != nil {
		return 0, err
	}
	return n, nil
}

type windowGeometry struct {
	x, y int
	w, h int
}

func (g windowGeometry) valid() bool {
	return g.x >= 0 && g.y >= 0 && g.w > 0 && g.h > 0
}

type guiEbiten struct {
	g    *gui.GUI
	geom windowGeometry

	started bool
	endGui  chan bool

	state gui.State

	main    *ebiten.Image
	overlay *ebiten.Image
	prev    *ebiten.Image
	prevID  string
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

	// state of the left analogue stick of the first gamepad
	gamepadAnalogue [2]float64
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
	err := eg.inputKeyboard()
	if err != nil {
		return ebiten.Termination
	}
	err = eg.inputGamepad()
	if err != nil {
		return ebiten.Termination
	}
	err = eg.inputGamepadAxis()
	if err != nil {
		return ebiten.Termination
	}

	// drag and drop of files is a special type of input
	err = eg.inputDragAndDrop()
	if err != nil {
		logger.Log(logger.Allow, "gui", err.Error())
	}

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
			if s.Read != nil {
				if eg.audio.p != nil {
					err := eg.audio.p.Close()
					if err != nil {
						return fmt.Errorf("ebiten: %w", err)
					}
				}

				ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
					SampleRate:   s.Freq,
					ChannelCount: 2,
					Format:       oto.FormatSignedInt16LE,
				})
				if err != nil {
					return fmt.Errorf("ebiten: %w", err)
				}

				select {
				case <-ready:
					eg.audio.r = s.Read
					eg.audio.p = ctx.NewPlayer(&eg.audio)
					eg.audio.p.Play()
				case <-eg.endGui:
					return ebiten.Termination
				}
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

		if img.Main != nil {
			if eg.main == nil || eg.main.Bounds() != img.Main.Bounds() {
				eg.width = img.Main.Bounds().Dx()
				eg.height = img.Main.Bounds().Dy()
				eg.main = ebiten.NewImage(eg.width, eg.height)
			}
			eg.main.WritePixels(img.Main.Pix)
		}

		if img.Prev != nil {
			eg.prevID = img.ID
			if eg.prev == nil || eg.prev.Bounds() != img.Prev.Bounds() {
				eg.prev = ebiten.NewImage(eg.width, eg.height)
			}
			if img.ID != eg.prevID {
				eg.prev.WritePixels(img.Prev.Pix)
			}
		}

		if img.Overlay != nil {
			if eg.overlay == nil || eg.overlay.Bounds() != img.Overlay.Bounds() {
				eg.overlay = ebiten.NewImage(eg.width, eg.height)
			}
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

	eg.geom.x, eg.geom.y = ebiten.WindowPosition()
	eg.geom.w, eg.geom.h = ebiten.WindowSize()
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

	// wait for the first state change and a possible quit request
	select {
	case eg.state = <-g.State:
		eg.audio.setState(eg.state)
	case <-endGui:
		return nil
	}

	var err error

	eg.geom, err = onWindowOpen()
	if err != nil {
		logger.Log(logger.Allow, "gui", err.Error())
	}

	defer func() {
		err := onWindowClose(eg.geom)
		if err != nil {
			logger.Log(logger.Allow, "gui", err.Error())
			return
		}
	}()

	return ebiten.RunGame(eg)
}
