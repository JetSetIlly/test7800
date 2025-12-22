package tia

import (
	"fmt"

	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware/spec"
	"github.com/jetsetilly/test7800/hardware/tia/audio"
	"github.com/jetsetilly/test7800/logger"
)

type TIA struct {
	inpt [6]uint8
	aud  *audio.Audio
	buf  *audioBuffer

	// count number of clocks before retreiving sample
	sampleCount      int
	sampleCountLimit int
	sampleCountStep  int

	// interface to the riot
	riot riot

	// use stereo mixing for audio
	stereo bool
}

type Context interface {
	Spec() spec.Spec
	UseAudio() bool
	UseStereo() bool
	SampleRate() (int, bool)
}

type riot interface {
	Read(idx uint16) (uint8, error)
}

type limiter interface {
	Nudge()
}

func Create(ctx Context, g *gui.GUI, riot riot, limiter limiter) *TIA {
	tia := &TIA{
		aud:    audio.NewAudio(),
		stereo: ctx.UseStereo(),
	}

	if g.AudioSetup != nil {
		tia.buf = &audioBuffer{
			data:  make([]uint8, 0, 4096),
			limit: limiter,
		}

		// decide on sampling rate
		var freq int
		freq = int(ctx.Spec().HorizScan * audio.SamplesPerScanline)
		if f, ok := ctx.SampleRate(); ok {
			freq = f
			r := float64(f) / (ctx.Spec().HorizScan * audio.SamplesPerScanline)
			r = 56.0 / r
			tia.sampleCountLimit = int(r * 10)
			tia.sampleCountStep = 10
		}

		logger.Logf(logger.Allow, "TIA", "using sampling rate of %d", freq)

		// notify UI of audio requirements
		var audioSetup gui.AudioSetup
		if ctx.UseAudio() {
			audioSetup = gui.AudioSetup{
				Read: tia.AudioBuffer(),
				Freq: freq,
			}
		} else {
			go func() {
				for range 4 {
					limiter.Nudge()
				}
			}()
		}
		select {
		case g.AudioSetup <- audioSetup:
		default:
		}
	}
	return tia
}

func (tia *TIA) String() string {
	return fmt.Sprintf("%#v", tia.inpt)
}

func (tia *TIA) Reset() error {
	tia.inpt = [6]uint8{
		0x80, 0x80, 0x80, 0x80,
		0x80, 0x80,
	}
	return nil
}

func (tia *TIA) Insert(externalChips audio.SoundChipIterator) error {
	tia.aud.PiggybackExternalSound(externalChips)
	return nil
}

func (tia *TIA) AudioBuffer() gui.AudioReader {
	return tia.buf
}

func (tia *TIA) Label() string {
	return "TIA"
}

func (tia *TIA) Status() string {
	return tia.Label()
}

func (tia *TIA) Access(write bool, idx uint16, data uint8) (uint8, error) {
	if write {
		return data, tia.write(idx, data)
	}
	return tia.read(idx)
}

func (tia *TIA) read(idx uint16) (uint8, error) {
	switch idx {
	case 0x08:
		return tia.inpt[0], nil
	case 0x09:
		return tia.inpt[1], nil
	case 0x0a:
		return tia.inpt[2], nil
	case 0x0b:
		return tia.inpt[3], nil
	case 0x0c:
		return tia.inpt[4], nil
	case 0x0d:
		return tia.inpt[5], nil
	}
	return 0, nil
}

func (tia *TIA) write(idx uint16, data uint8) error {
	if tia.buf != nil {
		tia.buf.crit.Lock()
		defer tia.buf.crit.Unlock()
	}
	switch idx {
	case 0x15:
		tia.aud.Channel0.Registers.Control = data & 0x0f
	case 0x16:
		tia.aud.Channel1.Registers.Control = data & 0x0f
	case 0x17:
		tia.aud.Channel0.Registers.Freq = data & 0x1f
	case 0x18:
		tia.aud.Channel1.Registers.Freq = data & 0x1f
	case 0x19:
		tia.aud.Channel0.Registers.Volume = data & 0x0f
	case 0x1a:
		tia.aud.Channel1.Registers.Volume = data & 0x0f
	}
	return nil
}

func (tia *TIA) PortWrite(idx uint16, data uint8, mask uint8) error {
	switch idx {
	case 0x08:
		tia.inpt[0] = (tia.inpt[0] & mask) | (data & ^mask)
	case 0x09:
		tia.inpt[1] = (tia.inpt[1] & mask) | (data & ^mask)
	case 0x0a:
		tia.inpt[2] = (tia.inpt[2] & mask) | (data & ^mask)
	case 0x0b:
		tia.inpt[3] = (tia.inpt[3] & mask) | (data & ^mask)
	case 0x0c:
		tia.inpt[4] = (tia.inpt[4] & mask) | (data & ^mask)
	case 0x0d:
		tia.inpt[5] = (tia.inpt[5] & mask) | (data & ^mask)
	}
	return nil
}

func (tia *TIA) Tick() {
	if tia.buf == nil {
		return
	}

	sample := tia.aud.Step()

	// if sampling rate has been specified then we do our own count
	if tia.sampleCountLimit > 0 {
		tia.sampleCount += tia.sampleCountStep
		if tia.sampleCount > tia.sampleCountLimit {
			sample = true
			tia.sampleCount = 0
		} else {
			sample = false
		}
	}

	if sample {
		tia.buf.crit.Lock()
		defer tia.buf.crit.Unlock()

		if tia.stereo {
			v0, v1 := tia.aud.Stereo()
			tia.buf.data = append(tia.buf.data, uint8(v0), uint8(v0>>8))
			tia.buf.data = append(tia.buf.data, uint8(v1), uint8(v1>>8))
		} else {
			v := tia.aud.Mono()
			tia.buf.data = append(tia.buf.data, uint8(v), uint8(v>>8))
			tia.buf.data = append(tia.buf.data, uint8(v), uint8(v>>8))
		}
	}
}
