package tia

import (
	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware/memory/external"
	"github.com/jetsetilly/test7800/hardware/tia/audio"
)

type TIA struct {
	inpt [6]uint8
	aud  *audio.Audio
	buf  *audioBuffer

	// interface to the riot
	riot riot
}

type Context interface {
	IsAtari7800() bool
}

type riot interface {
	Read(idx uint16) (uint8, error)
}

type limiter interface {
	Nudge()
}

func Create(_ Context, g *gui.GUI, riot riot, limiter limiter) *TIA {
	tia := &TIA{
		aud: audio.NewAudio(),
	}
	if g.AudioSetup != nil {
		tia.buf = &audioBuffer{
			data:  make([]uint8, 0, 4096),
			limit: limiter,
		}
	}
	tia.Insert(external.CartridgeInsertor{}, nil)
	return tia
}

func (tia *TIA) Insert(c external.CartridgeInsertor, externalChips audio.SoundChipIterator) error {
	// https://forums.atariage.com/topic/127162-question-about-joysticks-and-how-they-are-read/#findComment-1537159
	if c.OneButtonStick {
		tia.inpt = [6]uint8{
			0x00, 0x00, 0x80, 0x80,
			0x80, 0x80,
		}
	} else {
		// inpt initialised as though two-button sticks/gamepads are being used. INPT4 and INPT5
		// represent the primary fire button and the high bit pulled high by default. the high bits
		// of INPT0 and INPT1 meanwhile, represents the secondary button and is held low by default.
		//
		// INPT2 and INPT3 are not connected for joystick peripherals and the high bit is held high
		// in this case
		tia.inpt = [6]uint8{
			0x00, 0x00, 0x00, 0x00,
			0x80, 0x80,
		}
	}

	// piggyback any external soundchips to the TIA audio
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
		tia.inpt[0] = (data & mask) | (data & ^mask)
	case 0x09:
		tia.inpt[1] = (data & mask) | (data & ^mask)
	case 0x0a:
		tia.inpt[2] = (data & mask) | (data & ^mask)
	case 0x0b:
		tia.inpt[3] = (data & mask) | (data & ^mask)
	case 0x0c:
		tia.inpt[4] = (data & mask) | (data & ^mask)
	case 0x0d:
		tia.inpt[5] = (data & mask) | (data & ^mask)
	}
	return nil
}

func (tia *TIA) Tick() {
	if tia.buf == nil {
		return
	}

	if tia.aud.Step() {
		tia.buf.crit.Lock()
		defer tia.buf.crit.Unlock()

		m := tia.aud.Mono()
		tia.buf.data = append(tia.buf.data, uint8(m), uint8(m>>8))
		tia.buf.data = append(tia.buf.data, uint8(m), uint8(m>>8))
	}
}
