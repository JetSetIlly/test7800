package tia

import (
	"sync"

	"github.com/jetsetilly/test7800/hardware/tia/audio"
	"github.com/jetsetilly/test7800/hardware/tia/audio/mix"
	"github.com/jetsetilly/test7800/ui"
)

type audioBuffer struct {
	crit sync.Mutex
	data []uint8
}

func (b *audioBuffer) Read(buf []uint8) (int, error) {
	b.crit.Lock()
	defer b.crit.Unlock()

	n := min(len(b.data), len(buf))
	copy(buf, b.data[:n])
	b.data = b.data[n:]

	// return zero bytes is problematic for the WASM build of the emulator. we
	// could get around this by returning a minimum of 4 bytes, however, this
	// can cause the audio to drift out of sync with the video if too many of
	// these are sent
	//
	// we could indicated just 1 byte but this means the sample data becomes
	// unaligned. the number of bytes returned needs to be a multiple of two
	// because of the sample format (2 channel, 16bit little-endian)
	//
	// https://github.com/ebitengine/oto/issues/261
	return n, nil
}

type TIA struct {
	mem  Memory
	aud  *audio.Audio
	buf  *audioBuffer
	inpt [6]uint8
}

type Memory interface {
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
}

type Context interface {
	IsAtari7800() bool
}

func Create(ctx Context, ui *ui.UI, mem Memory) *TIA {
	tia := &TIA{
		mem: mem,
		aud: audio.NewAudio(ctx.IsAtari7800()),
		buf: &audioBuffer{
			data: make([]uint8, 0, 4096),
		},

		// inpt initialised as though sticks are being used
		inpt: [6]uint8{
			0x00, 0x00, 0x00, 0x00,
			0x80, 0x80,
		},
	}
	ui.RegisterAudio <- tia.buf
	return tia
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
	case 0x15:
		return tia.aud.Channel0.Registers.Control, nil
	case 0x16:
		return tia.aud.Channel1.Registers.Control, nil
	case 0x17:
		return tia.aud.Channel0.Registers.Freq, nil
	case 0x18:
		return tia.aud.Channel1.Registers.Freq, nil
	case 0x19:
		return tia.aud.Channel0.Registers.Volume, nil
	case 0x1a:
		return tia.aud.Channel1.Registers.Volume, nil
	}
	return 0, nil
}

func (tia *TIA) Write(idx uint16, data uint8) error {
	switch idx {
	case 0x08:
		tia.inpt[0] = data
	case 0x09:
		tia.inpt[1] = data
	case 0x0a:
		tia.inpt[2] = data
	case 0x0b:
		tia.inpt[3] = data
	case 0x0c:
		tia.inpt[4] = data
	case 0x0d:
		tia.inpt[5] = data
	case 0x15:
		tia.aud.Channel0.Registers.Control = data
	case 0x16:
		tia.aud.Channel1.Registers.Control = data
	case 0x17:
		tia.aud.Channel0.Registers.Freq = data
	case 0x18:
		tia.aud.Channel1.Registers.Freq = data
	case 0x19:
		tia.aud.Channel0.Registers.Volume = data
	case 0x1a:
		tia.aud.Channel1.Registers.Volume = data
	}
	return nil
}

func (tia *TIA) Tick() {
	if tia.aud.Step() {
		m := mix.Mono(tia.aud.Vol0, tia.aud.Vol1)

		tia.buf.crit.Lock()
		defer tia.buf.crit.Unlock()

		tia.buf.data = append(tia.buf.data, uint8(m), uint8(m>>8))
		tia.buf.data = append(tia.buf.data, uint8(m), uint8(m>>8))
	}
}
