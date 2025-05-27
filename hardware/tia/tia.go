package tia

import (
	"github.com/jetsetilly/test7800/hardware/tia/audio"
	"github.com/jetsetilly/test7800/hardware/tia/audio/mix"
	"github.com/jetsetilly/test7800/ui"
)

type TIA struct {
	inpt     [6]uint8
	aud      *audio.Audio
	buf      *audioBuffer
	halfStep bool
}

type Context interface {
	IsAtari7800() bool
}

func Create(ctx Context, ui *ui.UI) *TIA {
	tia := &TIA{
		aud: audio.NewAudio(),

		// inpt initialised as though sticks are being used
		inpt: [6]uint8{
			0x00, 0x00, 0x00, 0x00,
			0x80, 0x80,
		},
	}
	if ui.AudioSetup != nil {
		tia.buf = &audioBuffer{
			tia:  tia,
			data: make([]uint8, 0, 4096),
		}
	}
	return tia
}

func (tia *TIA) AudioBuffer() ui.AudioReader {
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
		return data, tia.Write(idx, data)
	}
	return tia.Read(idx)
}

func (tia *TIA) Read(idx uint16) (uint8, error) {
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

func (tia *TIA) Write(idx uint16, data uint8) error {
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

	tia.halfStep = !tia.halfStep
	if tia.halfStep {
		return
	}

	tia.buf.crit.Lock()
	defer tia.buf.crit.Unlock()

	tia.tick()
}

func (tia *TIA) tick() bool {
	if !tia.aud.Step() {
		return false
	}

	m := mix.Mono(tia.aud.Vol0, tia.aud.Vol1)
	tia.buf.data = append(tia.buf.data, uint8(m), uint8(m>>8))
	tia.buf.data = append(tia.buf.data, uint8(m), uint8(m>>8))
	return true
}
