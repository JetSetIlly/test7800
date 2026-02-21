package tia

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware/spec"
	"github.com/jetsetilly/test7800/hardware/tia/audio"
	"github.com/jetsetilly/test7800/logger"
)

type Register int

const (
	VBLANK Register = 0x01
	WSYNC  Register = 0x02
	RSYNC  Register = 0x03
	INPT0  Register = 0x08
	INPT1  Register = 0x09
	INPT2  Register = 0x0a
	INPT3  Register = 0x0b
	INPT4  Register = 0x0c
	INPT5  Register = 0x0d
	AUDC0  Register = 0x15
	AUDC1  Register = 0x16
	AUDF0  Register = 0x17
	AUDF1  Register = 0x18
	AUDV0  Register = 0x19
	AUDV1  Register = 0x1a
)

type TIA struct {
	inpt [6]uint8
	aud  *audio.Audio
	buf  *audioBuffer

	// count number of clocks before retreiving sample
	sampleCount      int
	sampleCountLimit int
	sampleCountStep  int

	// use stereo mixing for audio
	stereo bool

	// tia registers
	vblank uint8
	wsync  bool
	rsync  bool

	pclk  int
	hsync int
}

type Context interface {
	Spec() spec.Spec
	UseAudio() bool
	UseStereo() bool
	SampleRate() (int, bool)
}

type Limiter interface {
	Nudge()
}

func Create(ctx Context, g *gui.ChannelsDebugger, limiter Limiter) *TIA {
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
	var s strings.Builder
	fmt.Fprintln(&s, "Left Player")
	fmt.Fprintf(&s, " INPT0: %#02x\n", tia.inpt[0])
	fmt.Fprintf(&s, " INPT1: %#02x\n", tia.inpt[1])
	fmt.Fprintf(&s, " INPT4: %#02x\n", tia.inpt[4])
	fmt.Fprintln(&s, "Right Player")
	fmt.Fprintf(&s, " INPT2: %#02x\n", tia.inpt[2])
	fmt.Fprintf(&s, " INPT3: %#02x\n", tia.inpt[3])
	fmt.Fprintf(&s, " INPT5: %#02x\n", tia.inpt[5])
	return strings.Trim(s.String(), "\n")
}

func (tia *TIA) Reset() error {
	tia.inpt = [6]uint8{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00,
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
		return data, tia.write(Register(idx), data)
	}
	return tia.read(Register(idx))
}

func (tia *TIA) read(reg Register) (uint8, error) {
	switch reg {
	case INPT0:
		return tia.inpt[0], nil
	case INPT1:
		return tia.inpt[1], nil
	case INPT2:
		return tia.inpt[2], nil
	case INPT3:
		return tia.inpt[3], nil
	case INPT4:
		return tia.inpt[4], nil
	case INPT5:
		return tia.inpt[5], nil
	}
	return 0, nil
}

func (tia *TIA) write(reg Register, data uint8) error {
	if tia.buf != nil {
		tia.buf.crit.Lock()
		defer tia.buf.crit.Unlock()
	}
	switch reg {
	case VBLANK:
		tia.vblank = data
	case WSYNC:
		tia.wsync = true
	case RSYNC:
		tia.rsync = true
	case AUDC0:
		tia.aud.Channel0.Registers.Control = data & 0x0f
	case AUDC1:
		tia.aud.Channel1.Registers.Control = data & 0x0f
	case AUDF0:
		tia.aud.Channel0.Registers.Freq = data & 0x1f
	case AUDF1:
		tia.aud.Channel1.Registers.Freq = data & 0x1f
	case AUDV0:
		tia.aud.Channel0.Registers.Volume = data & 0x0f
	case AUDV1:
		tia.aud.Channel1.Registers.Volume = data & 0x0f
	}
	return nil
}

// PortWrite connects peripherals to the RIOT via the player ports
func (tia *TIA) PortWrite(reg Register, data uint8, mask uint8) error {
	switch reg {
	case INPT0:
		tia.inpt[0] = (tia.inpt[0] & mask) | (data & ^mask)
	case INPT1:
		tia.inpt[1] = (tia.inpt[1] & mask) | (data & ^mask)
	case INPT2:
		tia.inpt[2] = (tia.inpt[2] & mask) | (data & ^mask)
	case INPT3:
		tia.inpt[3] = (tia.inpt[3] & mask) | (data & ^mask)
	case INPT4:
		tia.inpt[4] = (tia.inpt[4] & mask) | (data & ^mask)
	case INPT5:
		tia.inpt[5] = (tia.inpt[5] & mask) | (data & ^mask)
	}
	return fmt.Errorf("tia: not a port connected register: %v", reg)
}

func (tia *TIA) Tick() bool {
	tia.pclk++

	if tia.pclk >= 4 {
		tia.pclk = 0
	} else if tia.pclk == 2 {
		if tia.rsync {
			tia.rsync = false
			tia.hsync = 56
		} else {
			tia.hsync++
			if tia.hsync >= 57 {
				tia.hsync = 0
				tia.wsync = false
			}
		}
	} else {
		return !tia.wsync
	}

	if tia.buf == nil {
		return !tia.wsync
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

	return !tia.wsync
}

func (tia *TIA) PaddlesGrounded() bool {
	return tia.vblank&0x80 == 0x80
}
