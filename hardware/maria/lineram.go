package maria

type lineram struct {
	// lineram is implemented as a single line image. this isn't an exact
	// representation of lineram but it's workable and produces adequate results
	// for now
	lineram [2][clksVisible]lineEntry

	// the index into the lineram. the values should be 0 or 1 and not equal
	lineramRead  int
	lineramWrite int
}

func (l *lineram) initialise() {
	l.lineramRead = 0
	l.lineramWrite = 1
}

func (l *lineram) newScanline() {
	l.lineramRead, l.lineramWrite = l.lineramWrite, l.lineramRead
	for s := range clksVisible {
		l.lineram[l.lineramWrite][s].set = false
	}
}

func (l *lineram) read(x int) lineEntry {
	return l.lineram[l.lineramRead][x]
}

func (l *lineram) write(x int, palette uint8, idx uint8) {
	l.lineram[l.lineramWrite][x].set = true
	l.lineram[l.lineramWrite][x].palette = palette
	l.lineram[l.lineramWrite][x].idx = idx
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

// the index value to use in the lineEntry struct to indicate that the colour of
// the lineram entry should be the currently selected background colour. this
// happens when kangaroo mode is enabled
const bgIdx = 255
