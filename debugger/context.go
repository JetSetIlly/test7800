package debugger

import (
	"math/rand/v2"

	"github.com/jetsetilly/test7800/hardware/spec"
)

type context struct {
	console    string
	spec       string
	rand       *rand.Rand
	Breaks     []error
	useOverlay bool
	audio      string
	sampleRate int
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
	return ctx.audio != "NONE"
}

func (ctx *context) UseStereo() bool {
	return ctx.audio == "STEREO"
}

// returns false is the sample rate hasn't be specified
func (ctx *context) SampleRate() (int, bool) {
	return ctx.sampleRate, ctx.sampleRate > 0
}
