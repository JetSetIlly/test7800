package main

import (
	"math/rand/v2"
	"os"

	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/gui/ebiten"
	"github.com/jetsetilly/test7800/hardware"
	"github.com/jetsetilly/test7800/hardware/spec"
	"github.com/jetsetilly/test7800/logger"
)

type context struct {
	console    string
	spec       string
	rand       *rand.Rand
	Breaks     []error
	useOverlay bool
}

func (ctx *context) Spec() spec.Spec {
	switch ctx.spec {
	case "NTSC":
		return spec.NTSC
	case "PAL":
		return spec.PAL
	}
	panic("currently unsupported specification")
}

func (ctx *context) IsAtari7800() bool {
	return ctx.console == "7800"
}

func (ctx *context) Reset() {
	ctx.Breaks = ctx.Breaks[:0]
	ctx.rand = rand.New(rand.NewPCG(0, 0))
}

func (ctx *context) Rand8Bit() uint8 {
	return uint8(ctx.rand.IntN(255))
}

func (ctx *context) Rand16Bit() uint16 {
	return uint16(ctx.rand.IntN(65535))
}

func (ctx *context) Break(e error) {
	ctx.Breaks = append(ctx.Breaks, e)
}

func (ctx *context) UseOverlay() bool {
	return ctx.useOverlay
}

func (ctx *context) UseAudio() bool {
	return false
}

func (ctx *context) UseStereo() bool {
	return false
}

func (ctx *context) SampleRate() (int, bool) {
	return 48000, true
}

func (ctx *context) AllowLogging() bool {
	return true
}

func (ctx *context) Overscan() string {
	return "AUTO"
}

func main() {
	// logger messages will be viewable in javascript log for WASM build
	logger.SetEcho(os.Stderr, false)

	g := gui.NewChannels()

	// using PAL BIOS so we get asteroids for free
	ctx := context{
		console: "7800",
		spec:    "PAL",
	}
	ctx.Reset()

	con := hardware.Create(&ctx, g.Debugger())
	con.Reset(true, func() bool { return true })

	update := func() error {
		fn := con.MARIA.Coords.Frame
		for con.MARIA.Coords.Frame == fn {
			err := con.Step()
			if err != nil {
				return err
			}
		}
		return nil
	}

	// start off gui in the paused state. gui won't properly begin until it receives a state change
	g.State <- gui.StatePaused

	ebiten.Launch(nil, g.GUI(), update)
}
