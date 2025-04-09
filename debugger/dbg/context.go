package dbg

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

type Context struct {
	console string
	spec    string
	rand    *rand.Rand
	Breaks  []error
}

func Create(console string, spec string) (Context, error) {
	ctx := Context{
		console: strings.ToUpper(console),
		spec:    strings.ToUpper(spec),
	}

	if ctx.console != "7800" {
		return ctx, fmt.Errorf("unsupported console type: %s", ctx.console)
	}

	if !(ctx.spec == "NTSC" || ctx.spec == "PAL") {
		return ctx, fmt.Errorf("unsupported TV specification: %s", ctx.spec)
	}

	ctx.rand = rand.New(rand.NewPCG(0, 0))
	return ctx, nil
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
