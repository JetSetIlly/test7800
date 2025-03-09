package maria

import (
	"fmt"
	"strings"

	_ "embed"
)

type mariaCtrl struct {
	colourKill bool
	dma        int // 0 to 3 only
	charWidth  bool
	border     bool
	kanagroo   bool
	readMode   int // 0 to 3 only
}

func (ctrl *mariaCtrl) write(data uint8) {
	ctrl.colourKill = data&0x80 == 0x80
	ctrl.dma = int((data >> 5) & 0x03)
	ctrl.charWidth = data&0x10 == 0x10
	ctrl.border = data&0x80 == 0x80
	ctrl.kanagroo = data&0x40 == 0x40
	ctrl.readMode = int(data & 0x03)
}

func (ctrl *mariaCtrl) String() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("ck=%v ", ctrl.colourKill))
	s.WriteString(fmt.Sprintf("dma=%d ", ctrl.dma))
	s.WriteString(fmt.Sprintf("cw=%v ", ctrl.charWidth))
	s.WriteString(fmt.Sprintf("bc=%v ", ctrl.border))
	s.WriteString(fmt.Sprintf("km=%v ", ctrl.kanagroo))
	s.WriteString(fmt.Sprintf("rm=%d ", ctrl.readMode))
	return s.String()
}

type Maria struct {
	bg       uint8
	wsync    bool
	palette  [8][3]uint8
	dpph     uint8
	dppl     uint8
	charbase uint8
	offset   uint8
	ctrl     mariaCtrl
}

func (mar *Maria) Label() string {
	return "MARIA"
}

func (mar *Maria) Status() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("%s: bg=%#02x wsync=%v\n%s\ndpph=%#02x dppl=%#02x charbase=%#02x offset=%#02x",
		mar.Label(), mar.bg, mar.wsync,
		mar.ctrl.String(),
		mar.dpph, mar.dppl, mar.charbase, mar.offset,
	))
	for p := range mar.palette {
		s.WriteString("\n")
		s.WriteString(fmt.Sprintf("palette %d:", p))
		for c := range mar.palette[p] {
			s.WriteString(fmt.Sprintf(" %d=%#02x", c, mar.palette[p][c]))
		}
	}
	return s.String()
}

func (mar *Maria) Read(idx uint16) (uint8, error) {
	switch idx {
	case 0x020:
		return mar.bg, nil
	case 0x021:
		return mar.palette[0][0], nil
	case 0x022:
		return mar.palette[0][1], nil
	case 0x023:
		return mar.palette[0][2], nil
	case 0x024:
		// wsync
		// (write only)
		return 0, nil
	case 0x025:
		return mar.palette[1][0], nil
	case 0x026:
		return mar.palette[1][1], nil
	case 0x027:
		return mar.palette[1][2], nil
	case 0x028:
		// maria status
		// TODO: proper VBLANK status. bit 7 is true if VBLANK is enabled
		return 0x80, nil
	case 0x029:
		return mar.palette[2][0], nil
	case 0x02a:
		return mar.palette[2][1], nil
	case 0x02b:
		return mar.palette[2][2], nil
	case 0x02c:
		// display list point high
		// (write only)
		return 0, nil
	case 0x02d:
		return mar.palette[3][0], nil
	case 0x02e:
		return mar.palette[3][1], nil
	case 0x02f:
		return mar.palette[3][2], nil
	case 0x030:
		// display list point low
		// (write only)
		return 0, nil
	case 0x031:
		return mar.palette[4][0], nil
	case 0x032:
		return mar.palette[4][1], nil
	case 0x033:
		return mar.palette[4][2], nil
	case 0x034:
		// character base address
		// (write only)
		return 0, nil
	case 0x035:
		return mar.palette[5][0], nil
	case 0x036:
		return mar.palette[5][1], nil
	case 0x037:
		return mar.palette[5][2], nil
	case 0x038:
		// reserved for future expansion
		return mar.offset, nil
	case 0x039:
		return mar.palette[6][0], nil
	case 0x03a:
		return mar.palette[6][1], nil
	case 0x03b:
		return mar.palette[6][2], nil
	case 0x03c:
		// maria control
		// (write only)
		return 0, nil
	case 0x03d:
		return mar.palette[7][0], nil
	case 0x03e:
		return mar.palette[7][1], nil
	case 0x03f:
		return mar.palette[7][2], nil
	}

	panic("read: not a maria address")
}

func (mar *Maria) Write(idx uint16, data uint8) error {
	switch idx {
	case 0x020:
		mar.bg = data
	case 0x021:
		mar.palette[0][0] = data
	case 0x022:
		mar.palette[0][1] = data
	case 0x023:
		mar.palette[0][2] = data
	case 0x024:
		mar.wsync = true
	case 0x025:
		mar.palette[1][0] = data
	case 0x026:
		mar.palette[1][1] = data
	case 0x027:
		mar.palette[1][2] = data
	case 0x028:
		// maria status (read only)
	case 0x029:
		mar.palette[2][0] = data
	case 0x02a:
		mar.palette[2][1] = data
	case 0x02b:
		mar.palette[2][2] = data
	case 0x02c:
		// display list point high
		mar.dpph = data
	case 0x02d:
		mar.palette[3][0] = data
	case 0x02e:
		mar.palette[3][1] = data
	case 0x02f:
		mar.palette[3][2] = data
	case 0x030:
		// display list point low
		mar.dppl = data
	case 0x031:
		mar.palette[4][0] = data
	case 0x032:
		mar.palette[4][1] = data
	case 0x033:
		mar.palette[4][2] = data
	case 0x034:
		// character base
		mar.charbase = data
	case 0x035:
		mar.palette[5][0] = data
	case 0x036:
		mar.palette[5][1] = data
	case 0x037:
		mar.palette[5][2] = data
	case 0x038:
		// reserved for future expansion
		mar.offset = data
	case 0x039:
		mar.palette[6][0] = data
	case 0x03a:
		mar.palette[6][1] = data
	case 0x03b:
		mar.palette[6][2] = data
	case 0x03c:
		// maria control
		mar.ctrl.write(data)
	case 0x03d:
		mar.palette[7][0] = data
	case 0x03e:
		mar.palette[7][1] = data
	case 0x03f:
		mar.palette[7][2] = data
	default:
		panic("read: not a maria address")
	}

	return nil
}
