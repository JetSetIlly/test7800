package maria

import (
	"fmt"
	"strings"
)

type dl struct {
	// if the indirect bit is set then the size of the header is 5 bytes rather than 4
	indirect bool

	// writemode is not present in a 4 byte header
	writemode bool

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

func (dl *dl) Status() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("ct=%d origin=%04x\n", dl.ct, dl.origin))
	if dl.isEnd {
		s.WriteString("end")
		return s.String()
	}
	if dl.indirect {
		s.WriteString(fmt.Sprintf("indirect=%v writebit=%v\n", dl.indirect, dl.writemode))
	}
	s.WriteString(fmt.Sprintf("high=%02x ", dl.highAddress))
	s.WriteString(fmt.Sprintf("low=%02x\n", dl.lowAddress))
	s.WriteString(fmt.Sprintf("palette=%03b ", dl.palette))
	s.WriteString(fmt.Sprintf("width=%05b\n", dl.width))
	s.WriteString(fmt.Sprintf("pos=%02x\n", dl.horizontalPosition))
	return s.String()
}

func (mar *Maria) nextDL(reset bool) error {
	// the amount we adjust the DLL pointer by to the next entry in the DL
	// depends on the size of the previous DL header
	var prevSize int

	if reset {
		mar.DL.ct = 0
	} else {
		mar.DL.ct++
		if mar.DL.indirect {
			prevSize = 5
		} else {
			prevSize = 4
		}
	}

	mar.DL.origin = (uint16(mar.DLL.highAddress) << 8) | (uint16(mar.DLL.lowAddress) + uint16(mar.DL.ct*prevSize))

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
		mar.DL.indirect = false
		mar.DL.writemode = false
		mar.DL.lowAddress = 0
		mar.DL.highAddress = 0
		mar.DL.palette = 0
		mar.DL.width = 0
		mar.DL.horizontalPosition = 0
		return nil
	}

	// the size of the DL header is different for indirect and direct modes
	mar.DL.indirect = mode&0x5f == 0x40
	if mar.DL.indirect {
		// for indirect mode the header is 5bytes long

		// the write bit is also part of the second byte, along with the indirect bit
		mar.DL.writemode = mode&0x80 == 0x80

		mar.DL.highAddress, err = mar.mem.Read(mar.DL.origin + 2)
		if err != nil {
			return err
		}

		d, err := mar.mem.Read(mar.DL.origin + 3)
		if err != nil {
			return err
		}
		mar.DL.palette = (d & 0xe0) >> 5
		mar.DL.width = d & 0x1f
		mar.DL.width ^= 0x1f
		mar.DL.width += 1
		mar.DL.width &= 0x1f

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
		// for direct mode the header is 4bytes long

		// in direct mode the second byte forms the palette and width values.
		// we've already read the second byte into the mode variable
		mar.DL.palette = (mode & 0xe0) >> 5
		mar.DL.width = mode & 0x1f
		mar.DL.width ^= 0x1f
		mar.DL.width += 1
		mar.DL.width &= 0x1f

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
	offset      int   // 4 bits of first byte (we need a signed value for this)
	highAddress uint8 // second byte
	lowAddress  uint8 // third byte

	// meta information about the DLL. which number it is in the list and the
	// address from which it was loaded
	ct     int
	origin uint16
}

func (dll *dll) Status() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("ct=%d origin=%04x\n", dll.ct, dll.origin))
	s.WriteString(fmt.Sprintf("dli=%v ", dll.dli))
	s.WriteString(fmt.Sprintf("h16=%v ", dll.h16))
	s.WriteString(fmt.Sprintf("h8=%v\n", dll.h8))
	s.WriteString(fmt.Sprintf("offset=%d ", dll.offset))
	s.WriteString(fmt.Sprintf("high=%02x ", dll.highAddress))
	s.WriteString(fmt.Sprintf("low=%02x ", dll.lowAddress))
	return s.String()
}

func (mar *Maria) nextDLL(reset bool) (bool, error) {
	if reset {
		mar.DLL.ct = 0
	} else {
		mar.DLL.offset--
		if mar.DLL.offset >= 0 {
			return mar.DLL.dli && mar.DLL.offset == 0, nil
		}
		mar.DLL.ct++
	}

	mar.DLL.origin = (uint16(mar.dpph) << 8) | (uint16(mar.dppl) + uint16(mar.DLL.ct*3))

	d, err := mar.mem.Read(mar.DLL.origin)
	if err != nil {
		return false, err
	}
	mar.DLL.dli = d&0x80 == 0x80
	mar.DLL.h16 = d&0x40 == 0x40
	mar.DLL.h8 = d&0x20 == 0x20
	mar.DLL.offset = int(d & 0x0f)

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
