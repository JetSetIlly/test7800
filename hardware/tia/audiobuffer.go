package tia

import "sync"

// audioBuffer is an io.Reader implementation that forwards TIA audio generated
// data to something that can play it back (or store it, etc.)
type audioBuffer struct {
	limit limiter
	crit  sync.Mutex
	data  []uint8
}

func (b *audioBuffer) Nudge() {
	b.limit.Nudge()
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
	// we could use just 1 byte but this means the sample data becomes
	// unaligned. the number of bytes returned needs to be a multiple of two
	// because of the sample format (2 channel, 16bit little-endian)
	//
	// https://github.com/ebitengine/oto/issues/261
	//
	// the new tick method which ensures a minimum number of bytes solves this issue
	return n, nil
}
