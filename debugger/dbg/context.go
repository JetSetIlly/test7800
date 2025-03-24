package dbg

import (
	"github.com/jetsetilly/test7800/hardware/cpu/execution"
)

const (
	maxTraceLen  = 10
	maxRecentLen = 10
)

type Context struct {
	Breaks []string
	trace  int

	Trace  []execution.Result
	Recent []execution.Result
}

func (ctx *Context) Break(s string) {
	ctx.Breaks = append(ctx.Breaks, s)
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
