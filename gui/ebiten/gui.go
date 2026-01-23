package ebiten

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/logger"
	"github.com/jetsetilly/test7800/version"

	_ "embed"
)

type windowGeometry struct {
	x, y       int
	w, h       int
	fullScreen bool
}

func (g windowGeometry) valid() bool {
	return g.x >= 0 && g.y >= 0 && g.w > 0 && g.h > 0
}

//go:embed "Hack-Regular.ttf"
var fontHack []byte

type guiEbiten struct {
	g      *gui.ChannelsGUI
	endGui <-chan bool

	geom  windowGeometry
	state gui.State

	overlayFont text.Face

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

	// position of mouse cursor on last update
	mouseX, mouseY int
	mouseCaptured  bool

	// whether to show the info text
	showInfo bool

	// time of last frame
	lastFrame time.Time

	// optional function called by during the update loop
	update func() error
}

func (eg *guiEbiten) Update() error {
	// service requests (the SetImage request is serviced in this function below)
	select {
	case eg.state = <-eg.g.State:
		eg.audio.setState(eg.state)
	case <-eg.endGui:
		if eg.audio.p != nil {
			eg.audio.p.Close()
		}
		return ebiten.Termination
	case lastSelectedROM := <-eg.g.FileRequest:
		n, err := fileRequest(lastSelectedROM)
		if err != nil {
			logger.Log(logger.Allow, "gui", err.Error())
		}
		select {
		case eg.g.RequestedFile <- n:
		default:
		}
	case msg := <-eg.g.ErrorDialog:
		showError(msg)
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
	err = eg.inputMouse()
	if err != nil {
		return ebiten.Termination
	}

	// drag and drop of files is a special type of input
	err = eg.inputDragAndDrop()
	if err != nil {
		logger.Log(logger.Allow, "gui", err.Error())
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
					if eg.state == gui.StatePaused {
						eg.audio.p.Pause()
					} else {
						eg.audio.p.Play()
					}
				case <-eg.endGui:
					return ebiten.Termination
				}
			}
		default:
		}
	}

	// run option update function
	if eg.update != nil {
		err := eg.update()
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

		if img.Prev != nil && img.ID != eg.prevID {
			eg.prevID = img.ID
			if eg.prev == nil || eg.prev.Bounds() != img.Prev.Bounds() {
				width := img.Prev.Bounds().Dx()
				height := img.Prev.Bounds().Dy()
				eg.prev = ebiten.NewImage(width, height)
			}
			eg.prev.WritePixels(img.Prev.Pix)
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
	defer func() {
		if eg.showInfo {
			var opts text.DrawOptions
			opts.GeoM.Translate(10, 10)
			text.Draw(screen, fmt.Sprintf("%s", time.Since(eg.lastFrame)), eg.overlayFont, &opts)
		}
		eg.lastFrame = time.Now()
	}()

	const aspectBias = 0.93

	var scaling float64
	winRatio := float64(eg.geom.w) / float64(eg.geom.h)
	imageRatio := float64(eg.width) / float64(eg.height)
	if imageRatio < winRatio {
		scaling = float64(eg.geom.h) / float64(eg.height)
	} else {
		scaling = float64(eg.geom.w) / (float64(eg.width) * aspectBias)
	}

	scalingX := scaling * aspectBias
	scalingY := scaling

	translateX := (float64(eg.geom.w) - (float64(eg.width) * scalingX)) / 2
	translateY := (float64(eg.geom.h) - (float64(eg.height) * scalingY)) / 2

	if eg.main != nil {
		if eg.prev != nil {
			var op ebiten.DrawImageOptions
			op.GeoM.Scale(scalingX, scalingY)
			op.GeoM.Translate(translateX, translateY)
			op.Filter = ebiten.FilterPixelated
			op.ColorScale.SetR(0.2)
			op.ColorScale.SetG(0.2)
			op.ColorScale.SetB(0.2)
			op.ColorScale.SetA(1.0)
			screen.DrawImage(eg.prev, &op)
		}
		if eg.main != nil {
			var op ebiten.DrawImageOptions
			op.GeoM.Scale(scalingX, scalingY)
			op.GeoM.Translate(translateX, translateY)
			op.Filter = ebiten.FilterPixelated
			op.Blend = ebiten.BlendSourceOver
			screen.DrawImage(eg.main, &op)
		}
		if eg.overlay != nil {
			var op ebiten.DrawImageOptions
			op.GeoM.Scale(scalingX, scalingY)
			op.GeoM.Translate(translateX, translateY)
			op.Filter = ebiten.FilterPixelated
			op.Blend = ebiten.BlendLighter
			screen.DrawImage(eg.overlay, &op)
		}

		eg.cursorFrame++

		// draw cursor if emulation is paused
		if eg.state == gui.StatePaused {
			v := uint8((math.Sin(float64(eg.cursorFrame/10))*0.5 + 0.5) * 255)
			screen.Set(eg.cursor[0], eg.cursor[1], color.RGBA{R: v, G: v, B: v, A: 255})
			screen.Set(eg.cursor[0]+1, eg.cursor[1], color.RGBA{R: v, G: v, B: v, A: 255})
			screen.Set(eg.cursor[0], eg.cursor[1]+1, color.RGBA{R: v, G: v, B: v, A: 255})
			screen.Set(eg.cursor[0]+1, eg.cursor[1]+1, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
}

func (eg *guiEbiten) Layout(width, height int) (int, int) {
	eg.geom.x, eg.geom.y = ebiten.WindowPosition()
	eg.geom.w = width
	eg.geom.h = height
	return width, height
}

func Launch(endGui <-chan bool, g *gui.ChannelsGUI, update func() error) error {
	ebiten.SetWindowTitle(version.Title())
	ebiten.SetVsyncEnabled(true)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowPosition(10, 10)
	ebiten.SetTPS(ebiten.SyncWithFPS)

	s, err := text.NewGoTextFaceSource(bytes.NewReader(fontHack))
	if err != nil {
		log.Fatal(err)
	}

	var baseFont text.Face = &text.GoTextFace{
		Source: s,
		Size:   15,
	}

	eg := &guiEbiten{
		g:           g,
		overlayFont: baseFont,
		endGui:      endGui,
		state:       gui.StateRunning,
		audio: audioPlayer{
			state: gui.StatePaused,
		},
		lastFrame: time.Now(),
		update:    update,
	}

	// loop to service requests until the first state change. (the main service loop is in the
	// Update() function)
	done := false
	for !done {
		select {
		case eg.state = <-g.State:
			// audio player is not ready yet so we don't really need to push the state change as we
			// do in the main update loop. however, we do so anyway in case something changes above
			// and we forget about this. there's no harm or performance penalty in setting the state
			// like this
			eg.audio.setState(eg.state)
			done = true
		case <-endGui:
			return nil
		case lastSelectedROM := <-g.FileRequest:
			n, err := fileRequest(lastSelectedROM)
			if err != nil {
				logger.Log(logger.Allow, "gui", err.Error())
			}
			select {
			case g.RequestedFile <- n:
			default:
			}
		case msg := <-eg.g.ErrorDialog:
			showError(msg)
		}
	}

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
