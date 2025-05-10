package debugger

import (
	"math/rand/v2"
)

type context struct {
	console    string
	spec       string
	rand       *rand.Rand
	Breaks     []error
	useOverlay bool
}

func (ctx *context) Spec() string {
	return ctx.spec
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
