package debugger

import (
	"math/rand/v2"

	"github.com/jetsetilly/test7800/hardware/spec"
)

type context struct {
	console       string
	requestedSpec string
	loaderSpec    string
	rand          *rand.Rand
	breaks        []error
	useOverlay    bool
	audio         string
	sampleRate    int
	overscan      string
}

func (ctx *context) Spec() spec.Spec {
	if ctx.requestedSpec == "AUTO" {
		switch ctx.loaderSpec {
		case "NTSC":
			return spec.NTSC
		case "PAL":
			return spec.PAL
		}
	}

	switch ctx.requestedSpec {
	case "AUTO", "NTSC":
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
	ctx.breaks = ctx.breaks[:0]
	ctx.rand = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
}

func (ctx *context) Rand8Bit() uint8 {
	return uint8(ctx.rand.IntN(255))
}

func (ctx *context) Rand16Bit() uint16 {
	return uint16(ctx.rand.IntN(65535))
}

func (ctx *context) Break(e error) {
	ctx.breaks = append(ctx.breaks, e)
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

func (ctx *context) Overscan() string {
	return ctx.overscan
}
