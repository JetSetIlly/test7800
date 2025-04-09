package dbg

import (
	"math/rand/v2"

	"github.com/jetsetilly/test7800/hardware/cpu/execution"
)

const (
	maxTraceLen  = 10
	maxRecentLen = 10
)

type Context struct {
	Recent []execution.Result

	Breaks []error

	trace int
	Trace []execution.Result

	rand *rand.Rand
}

func Create() Context {
	var ctx Context
	ctx.rand = rand.New(rand.NewPCG(0, 0))
	return ctx
}

func (ctx *Context) IsAtari7800() bool {
	return true
}

func (ctx *Context) Reset() {
	ctx.Breaks = ctx.Breaks[:0]
	ctx.trace = 0
	ctx.Trace = ctx.Trace[:0]
	ctx.Recent = ctx.Recent[:0]
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

// start trace specifying desired length. a length of -1 means trace until
// EndTrace() is called
func (ctx *Context) StartTrace(length int) {
	if length < 0 {
		length = -1
	}
	if length > maxTraceLen {
		length = maxTraceLen
	}
	ctx.Trace = ctx.Trace[:0]
	ctx.trace = length
}

func (ctx *Context) AddTrace(r execution.Result) {
	if ctx.trace != 0 {
		ctx.Trace = append(ctx.Trace, r)
		if ctx.trace > 0 {
			ctx.trace--
		}
	}
}

func (ctx *Context) EndTrace() {
	ctx.trace = 0
}

func (ctx *Context) AddRecent(r execution.Result) {
	ctx.Recent = append(ctx.Recent, r)
	if len(ctx.Recent) > maxRecentLen {
		ctx.Recent = ctx.Recent[1:]
	}
}
