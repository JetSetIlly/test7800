package maria

import (
	"fmt"
	"strings"
)

type dl struct {
	// if the indirect bit is set then the size of the header is 5 bytes rather than 4 bytes
	long bool

	// writemode and indirect are not present in a 4 byte header
	//
	// but writemode is only changed by a 5 byte DL header. ie. it's value
	// persists until it is explicitly changed
	indirect  bool
	writemode bool

	// these fields are common to both the 4 and 5 byte header
	lowAddress         uint8
	highAddress        uint8
	palette            uint8
	width              uint8
	horizontalPosition uint8

	// "If the second byte of a header is zero, it indicates the end of the
	// Display List, and DMA will stop allowing the 6502 to continue processing"
	isEnd bool

	// meta information about the DL. which number it is in the list and the
	// address from which it was loaded
	ct     int
	origin uint16
}

func (l *dl) Status() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("ct=%d origin=%04x\n", l.ct, l.origin))
	if l.isEnd {
		s.WriteString("end")
		return s.String()
	}
	if l.indirect {
		s.WriteString(fmt.Sprintf("indirect=%v writebit=%v\n", l.indirect, l.writemode))
	}
	s.WriteString(fmt.Sprintf("high=%02x ", l.highAddress))
	s.WriteString(fmt.Sprintf("low=%02x\n", l.lowAddress))
	s.WriteString(fmt.Sprintf("palette=%03b ", l.palette))
	s.WriteString(fmt.Sprintf("width=%05b\n", l.width))
	s.WriteString(fmt.Sprintf("pos=%02x", l.horizontalPosition))
	return s.String()
}

func (mar *Maria) nextDL(reset bool) error {
	// the amount we adjust the DLL pointer by to the next entry in the DL
	// depends on the size of the previous DL header
	var prevSize int

	if reset {
		mar.DL.ct = 0
		mar.DL.origin = (uint16(mar.DLL.highAddress) << 8) | uint16(mar.DLL.lowAddress)
	} else {
		mar.DL.ct++
		if mar.DL.long {
			prevSize = 5
		} else {
			prevSize = 4
		}
		mar.DL.origin += uint16(prevSize)
	}

	var err error

	mar.DL.lowAddress, err = mar.mem.Read(mar.DL.origin)
	if err != nil {
		return err
	}

	// second byte controls whether the display list is direct or indirect and
	// also whether this is the end of the DLL
	mode, err := mar.mem.Read(mar.DL.origin + 1)
	if err != nil {
		return err
	}
	mar.DL.isEnd = mode == 0x00

	// return early if this the end of the DLL
	if mar.DL.isEnd {
		mar.DL.long = false
		mar.DL.indirect = false
		mar.DL.lowAddress = 0
		mar.DL.highAddress = 0
		mar.DL.palette = 0
		mar.DL.width = 0
		mar.DL.horizontalPosition = 0

		// note that we're not reset the writemode field, which only changes
		// when specified by a 5 byte DL header

		return nil
	}

	setWidth := func(v uint8) {
		// width is two's complement
		mar.DL.width = v ^ 0x1f
		mar.DL.width += 1
		mar.DL.width &= 0x1f
	}

	// check if 4 or 5 byte header
	mar.DL.long = mode&0x5f == 0x40
	if mar.DL.long {
		// the write bit is also part of the second byte, along with the indirect bit
		mar.DL.writemode = mode&0x80 == 0x80
		mar.DL.indirect = mode&0x20 == 0x20

		mar.DL.highAddress, err = mar.mem.Read(mar.DL.origin + 2)
		if err != nil {
			return err
		}

		// palette and width are both contained in the second third byte
		d, err := mar.mem.Read(mar.DL.origin + 3)
		if err != nil {
			return err
		}

		mar.DL.palette = (d & 0xe0) >> 5
		setWidth(d & 0x1f)

		// "There is an added bonus to five byte headers. Because the end of DMA is
		// indicated by the presence of a zero in the second byte of a header, and
		// in a five byte header the width byte is not the second but the fourth, a
		// width of zero is valid in an extended header, and will be interpreted as
		// a value of 32"
		if mar.DL.width == 0 {
			// the way we do this is to set the value to 32. in real hardware I
			// think what really happens is that the five bit number wraps
			// around and it's 32 decrements until it reaches zero again. but
			// we're using an 8bit value so that won't work in quite the same way
			mar.DL.width = 32
		}

		mar.DL.horizontalPosition, err = mar.mem.Read(mar.DL.origin + 4)
		if err != nil {
			return err
		}
	} else {
		// for direct mode the header is 4 bytes long
		mar.DL.indirect = false

		// the value of writemode remains unchanged. this is good because it
		// means for direct gfx modes, only 4 byte headers need to be used once
		// the writemode bit has been set

		// in direct mode the second byte forms the palette and width values.
		// we've already read the second byte into the mode variable
		mar.DL.palette = (mode & 0xe0) >> 5
		setWidth(mode & 0x1f)

		mar.DL.highAddress, err = mar.mem.Read(mar.DL.origin + 2)
		if err != nil {
			return err
		}

		mar.DL.horizontalPosition, err = mar.mem.Read(mar.DL.origin + 3)
		if err != nil {
			return err
		}
	}

	return nil
}

type dll struct {
	dli         bool
	h16         bool
	h8          bool
	offset      uint8 // 4 bits of first byte (we need a signed value for this)
	highAddress uint8 // second byte
	lowAddress  uint8 // third byte

	// meta information about the DLL. which number it is in the list and the
	// address from which it was loaded
	ct     int
	origin uint16

	// working offset is an integer because we want to use negative values
	workingOffset int
}

func (l *dll) ID() string {
	return fmt.Sprintf("ct=%d origin=%04x\n", l.ct, l.origin)
}

func (l *dll) Status() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("ct=%d origin=%04x\n", l.ct, l.origin))
	s.WriteString(fmt.Sprintf("dli=%v ", l.dli))
	s.WriteString(fmt.Sprintf("h16=%v ", l.h16))
	s.WriteString(fmt.Sprintf("h8=%v\n", l.h8))
	s.WriteString(fmt.Sprintf("offset=%02x/%02x ", int(l.offset)-l.workingOffset, l.offset))
	s.WriteString(fmt.Sprintf("high=%02x ", l.highAddress))
	s.WriteString(fmt.Sprintf("low=%02x ", l.lowAddress))
	return s.String()
}

func (l *dll) inHole(a uint16) bool {
	// "Holey DMA has been aimed at 8 or 16 raster zones, but will have the same
	// effect for other zone sizes. MARIA can be told to interpret odd 4K blocks as
	// zeros, for 16 high zones, or odd 2K blocks as zeros for 8 high zones. This
	// will only work for addresses above '0x8000'"
	if a > 0x8000 {
		if l.h16 && (a&0x9000 == 0x9000) {
			return true
		}
		if l.h8 && (a&0x8800 == 0x8800) {
			return true
		}
	}
	return false
}

func (mar *Maria) nextDLL(reset bool) (bool, error) {
	if reset {
		mar.DLL.ct = 0
	} else {
		// "Included in each entry is a value called OFFSET, which indicates how many
		// rasters should use the specified Display List. OFFSET is decremented at the
		// end of each raster until it becomes negative, which indicates that the next
		// DLL entry should now be read and used."
		//
		// to implement this we're using a second field separate from the original
		// value. this is so we can display the original value if we need to via the
		// Status() function; and also so we can work with negative values which is
		// clearer than dealing with underflowed uint8
		mar.DLL.workingOffset--

		if mar.DLL.workingOffset > 0 {
			return false, nil
		}

		// "One of the bits of a DLL entry tells MARIA to generate a Display List
		// Interrupt (DLI) for that zone. The interrupt will actually occur
		// following DMA on the last line of the PREVIOUS zone."
		if mar.DLL.workingOffset == 0 {
			preview := mar.DLL.origin + uint16(3)
			d, err := mar.mem.Read(preview)
			if err != nil {
				return false, err
			}
			return d&0x80 == 0x80, nil
		}

		// workingOffset is less than zero so we read the next DLL
		mar.DLL.ct++
	}

	mar.DLL.origin = (uint16(mar.dpph) << 8) | uint16(mar.dppl)
	mar.DLL.origin += uint16(mar.DLL.ct * 3)

	d, err := mar.mem.Read(mar.DLL.origin)
	if err != nil {
		return false, err
	}

	mar.DLL.dli = d&0x80 == 0x80
	mar.DLL.h16 = d&0x40 == 0x40
	mar.DLL.h8 = d&0x20 == 0x20
	mar.DLL.offset = d & 0x0f

	// working offset is an integer because we want to use negative values
	mar.DLL.workingOffset = int(mar.DLL.offset)

	mar.DLL.highAddress, err = mar.mem.Read(mar.DLL.origin + 1)
	if err != nil {
		return false, err
	}

	mar.DLL.lowAddress, err = mar.mem.Read(mar.DLL.origin + 2)
	if err != nil {
		return false, err
	}

	return false, nil
}
