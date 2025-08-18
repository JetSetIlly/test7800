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
	lowAddress  uint8
	highAddress uint8
	palette     uint8
	width       uint8

	// note that the horizontal position for 320 modes are doubled by the Maria
	// when writing to line ram. this gives an effective resolution of 160
	// pixels.
	//
	// also common to 4 and 5 byte header
	horizontalPosition uint8

	// "If the second byte of a header is zero, it indicates the end of the
	// Display List, and DMA will stop allowing the 6502 to continue processing"
	isEnd bool

	// meta information about the DL. which number it is in the list and the
	// address from which it was loaded
	ct     int
	origin uint16
}

func (l *dl) ID() string {
	return fmt.Sprintf("ct=%d origin=%04x\n", l.ct, l.origin)
}

func (l *dl) String() string {
	return l.Status()
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

	// we read the second byte in the header first because it controls whether the other fields
	// contain meaningful information

	// second byte controls whether the display list is direct or indirect and
	// also whether this is the end of the DLL
	mode, err := mar.mem.Read(mar.DL.origin + 1)
	if err != nil {
		return err
	}

	// "Maria ignores bits 7 and 5, so your terminator check should be for '!(value & 0x5F)'"
	// https://7800.8bitdev.org/index.php/Common_Emulator_Development_Issues
	mar.DL.isEnd = mode&0x5f == 0x00

	// return early if this the end of the DLL
	if mar.DL.isEnd {
		mar.DL.long = false
		mar.DL.indirect = false
		mar.DL.lowAddress = 0
		mar.DL.highAddress = 0
		mar.DL.palette = 0
		mar.DL.width = 0
		mar.DL.horizontalPosition = 0

		// note that we're do not reset the writemode field, which only changes
		// when specified by a 5 byte DL header

		return nil
	}

	mar.DL.lowAddress, err = mar.mem.Read(mar.DL.origin)
	if err != nil {
		return err
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
	offset      uint8 // 4 bits of first byte
	highAddress uint8 // second byte
	lowAddress  uint8 // third byte

	// meta information about the DLL. which number it is in the list and the
	// address from which it was loaded
	ct     int
	origin uint16

	// "Included in each entry is a value called OFFSET, which indicates how many
	// rasters should use the specified Display List. OFFSET is decremented at the
	// end of each raster until it becomes negative, which indicates that the next
	// DLL entry should now be read and used."
	//
	// rather than adjustng the offset field, we instead adjust this workingOffset
	// field. this allows us to present the original value for debugging purposes
	workingOffset uint8
}

func (l *dll) ID() string {
	return fmt.Sprintf("ct=%d origin=%04x\n", l.ct, l.origin)
}

func (l *dll) String() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("ct=%d origin=%04x\n", l.ct, l.origin))
	s.WriteString(fmt.Sprintf("dli=%v ", l.dli))
	s.WriteString(fmt.Sprintf("h16=%v ", l.h16))
	s.WriteString(fmt.Sprintf("h8=%v\n", l.h8))
	s.WriteString(fmt.Sprintf("offset=%02x ", l.offset))
	s.WriteString(fmt.Sprintf("high=%02x ", l.highAddress))
	s.WriteString(fmt.Sprintf("low=%02x ", l.lowAddress))
	return s.String()
}

// Status is similar to String() but includes the current working offset along
// with the offset value in the DLL data
func (l *dll) Status() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("ct=%d origin=%04x\n", l.ct, l.origin))
	s.WriteString(fmt.Sprintf("dli=%v ", l.dli))
	s.WriteString(fmt.Sprintf("h16=%v ", l.h16))
	s.WriteString(fmt.Sprintf("h8=%v\n", l.h8))
	s.WriteString(fmt.Sprintf("offset=%02x/%02x ", l.offset-l.workingOffset, l.offset))
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
		if l.h16 && (a&0x1000 == 0x1000) {
			return true
		}
		if l.h8 && (a&0x0800 == 0x0800) {
			return true
		}
	}
	return false
}

func (mar *Maria) nextDLL(reset bool) (bool, error) {
	if reset {
		mar.DLL.ct = 0

		// "Once DMA is on, DPPH and DPPL may be written at any time, as they are only read at
		// thebeginning of the screen."
		mar.DLL.origin = (uint16(mar.dpph) << 8) | uint16(mar.dppl)
	} else {
		// offset is an unsigned integer between 0 and 15 and is the "height of
		// the zone, minus one". the DLL has expired therefore when the value is
		// less than zero, or in unsigned terms when the value is 0xff. however,
		// we also know that offset can never be greater than 0x0f so we actually
		// test for the working offset being less then 0x0f
		mar.DLL.workingOffset--
		if mar.DLL.workingOffset <= 0x0f {
			return false, nil
		}
		mar.DLL.ct++
		mar.DLL.origin += 3
	}

	d, err := mar.mem.Read(mar.DLL.origin)
	if err != nil {
		return true, err
	}

	mar.DLL.dli = d&0x80 == 0x80
	mar.DLL.h16 = d&0x40 == 0x40
	mar.DLL.h8 = d&0x20 == 0x20
	mar.DLL.offset = d & 0x0f
	mar.DLL.workingOffset = mar.DLL.offset

	mar.DLL.highAddress, err = mar.mem.Read(mar.DLL.origin + 1)
	if err != nil {
		return true, err
	}

	mar.DLL.lowAddress, err = mar.mem.Read(mar.DLL.origin + 2)
	if err != nil {
		return true, err
	}

	return true, nil
}
