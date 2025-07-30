package main

import (
	"math/rand/v2"
	"os"

	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/gui/ebiten"
	"github.com/jetsetilly/test7800/hardware"
	"github.com/jetsetilly/test7800/logger"
)

type Context struct {
	console    string
	spec       string
	rand       *rand.Rand
	Breaks     []error
	useOverlay bool
}

func (ctx *Context) Spec() string {
	return ctx.spec
}

func (ctx *Context) IsAtari7800() bool {
	return ctx.console == "7800"
}

func (ctx *Context) Reset() {
	ctx.Breaks = ctx.Breaks[:0]
	ctx.rand = rand.New(rand.NewPCG(0, 0))
}

func (ctx *Context) Rand8Bit() uint8 {
	return uint8(ctx.rand.IntN(255))
}

func (ctx *Context) Rand16Bit() uint16 {
	return uint16(ctx.rand.IntN(65535))
}

func (ctx *Context) Break(e error) {
	ctx.Breaks = append(ctx.Breaks, e)
}

func (ctx *Context) UseOverlay() bool {
	return ctx.useOverlay
}

// there is a problem with ebiten audio in the context of wasm so we launch
// without audio for now
const useAudio = false

func main() {
	// logger messages will be viewable in javascript log for WASM build
	logger.SetEcho(os.Stderr, false)

	g := gui.NewGUI()
	if useAudio {
		g = g.WithAudio()
	}

	ctx := Context{
		console:    "7800",
		spec:       "PAL",
		useOverlay: false,
	}
	ctx.Reset()

	con := hardware.Create(&ctx, g)
	con.Reset(true)

	g.UpdateGUI = func() error {
		fn := con.MARIA.Coords.Frame
		for con.MARIA.Coords.Frame == fn {
			err := con.Step()
			if err != nil {
				return err
			}
		}
		return nil
	}

	ebiten.Launch(nil, g)
}
