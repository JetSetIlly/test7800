package maria

import (
	"github.com/jetsetilly/test7800/hardware/spec"
)

type lineram struct {
	// lineram is implemented as a single line image. this isn't an exact
	// representation of lineram but it's workable and produces adequate results
	// for now
	lineram [2][spec.ClksVisible]lineEntry

	// the index into the lineram. the values should be 0 or 1 and not equal
	readIdx  int
	writeIdx int
}

func (l *lineram) initialise() {
	l.readIdx = 0
	l.writeIdx = 1
}

func (l *lineram) newScanline() {
	l.readIdx, l.writeIdx = l.writeIdx, l.readIdx
	for s := range spec.ClksVisible {
		l.lineram[l.writeIdx][s].set = false
	}
}

func (l *lineram) read(x int) lineEntry {
	return l.lineram[l.readIdx][x]
}

func (l *lineram) write(x int, palette uint8, idx uint8) {
	l.lineram[l.writeIdx][x].set = true
	l.lineram[l.writeIdx][x].palette = palette
	l.lineram[l.writeIdx][x].idx = idx
}

// each entry in lineram is a lineEntry instance
type lineEntry struct {
	// whether this entry is set or if the background should be shown
	set bool

	// which maria palette to index the colour from. values 0 to 7
	palette uint8

	// which entry in the specified palette to use. values 0 to 3
	idx uint8

	// * the data in the palette and idx fields are treated slightly differently
	// in 320B/D modes
}
