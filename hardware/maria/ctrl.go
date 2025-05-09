package maria

import (
	"fmt"
	"strings"
)

type mariaCtrl struct {
	colourKill bool
	dma        int // 0 to 3 only
	charWidth  bool
	border     bool
	kangaroo   bool
	readMode   int // 0 to 3 only
}

func (ctrl *mariaCtrl) reset() {
	ctrl.colourKill = false
	ctrl.dma = 3
	ctrl.charWidth = false
	ctrl.border = false
	ctrl.kangaroo = false
	ctrl.readMode = 0
}

func (ctrl *mariaCtrl) write(data uint8) {
	ctrl.colourKill = data&0x80 == 0x80
	ctrl.dma = int((data >> 5) & 0x03)
	ctrl.charWidth = data&0x10 == 0x10
	ctrl.border = data&0x08 == 0x08
	ctrl.kangaroo = data&0x04 == 0x04
	ctrl.readMode = int(data & 0x03)
}

func (ctrl *mariaCtrl) String() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("ck=%v ", ctrl.colourKill))
	s.WriteString(fmt.Sprintf("dma=%#02b ", ctrl.dma))
	s.WriteString(fmt.Sprintf("cw=%v ", ctrl.charWidth))
	s.WriteString(fmt.Sprintf("bc=%v ", ctrl.border))
	s.WriteString(fmt.Sprintf("km=%v ", ctrl.kangaroo))
	s.WriteString(fmt.Sprintf("rm=%#02b ", ctrl.readMode))
	return s.String()
}
