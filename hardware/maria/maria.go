package maria

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"strings"
)

type mariaCtrl struct {
	colourKill bool
	dma        int // 0 to 3 only
	charWidth  bool
	border     bool
	kanagroo   bool
	readMode   int // 0 to 3 only
}

func (ctrl *mariaCtrl) reset() {
	ctrl.colourKill = false
	ctrl.dma = 0
	ctrl.charWidth = false
	ctrl.border = false
	ctrl.kanagroo = false
	ctrl.readMode = 0
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
	s.WriteString(fmt.Sprintf("dma=%#02b ", ctrl.dma))
	s.WriteString(fmt.Sprintf("cw=%v ", ctrl.charWidth))
	s.WriteString(fmt.Sprintf("bc=%v ", ctrl.border))
	s.WriteString(fmt.Sprintf("km=%v ", ctrl.kanagroo))
	s.WriteString(fmt.Sprintf("rm=%#02b ", ctrl.readMode))
	return s.String()
}

var WarningErr = errors.New("warning")

type Maria struct {
	bg       uint8
	wsync    bool
	palette  [8][3]uint8
	dpph     uint8
	dppl     uint8
	charbase uint8
	offset   uint8
	ctrl     mariaCtrl

	// current DLL
	DLL dll
	DL  dl

	// read-only registers
	mstat uint8 // bit 7 is true if VBLANK is enabled

	// interface to console memory
	mem Memory

	// the current coordinates of the TV image
	Coords coords

	// any error from the most recent tick. errors are caused when the maria
	// can't continue in a normal fashion. for example, if dpph/dppl do not
	// point to a valid RAM area
	Error error

	// pixels for current frame
	currentFrame *image.RGBA
	rendering    chan *image.RGBA

	// the selected rgba palette to use when rendering the screen
	rgba [256]color.RGBA

	// creating a new image depends on the tv specification. the newImage()
	// function returns an appropriately sized image for the specification
	newImage func() *image.RGBA
}

type Memory interface {
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
}

func Create(mem Memory, spec string, rendering chan *image.RGBA) *Maria {
	mar := &Maria{
		mem:       mem,
		rendering: rendering,
	}

	switch strings.ToUpper(spec) {
	case "NTSC":
		mar.rgba = ntscPalette
		mar.newImage = func() *image.RGBA {
			return image.NewRGBA(image.Rect(0, 0, clksVisible, ntscVisibleBottom-ntscVisibleTop))
		}
	case "PAL":
		mar.rgba = palPalette
		mar.newImage = func() *image.RGBA {
			return image.NewRGBA(image.Rect(0, 0, clksVisible, palVisibleBottom-palVisibleTop))
		}
	default:
		panic("currently unsupported specification")
	}

	mar.currentFrame = mar.newImage()
	return mar
}

func (mar *Maria) Reset() {
	mar.Coords.Reset()
	mar.bg = 0
	mar.wsync = false
	mar.dpph = 0
	mar.dppl = 0
	mar.charbase = 0
	mar.offset = 0
	mar.mstat = 0
	mar.ctrl.reset()
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
	if mar.Error != nil {
		if errors.Is(mar.Error, WarningErr) {
			s.WriteString("warning: ")
		} else {
			s.WriteString("error: ")
		}
		s.WriteString(mar.Error.Error())
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
		return mar.mstat, nil
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
		// reserved for future expansion. this should always be zero
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
	return 0x00, fmt.Errorf("not a maria address (%#04x)", idx)
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
		// reserved for future expansion. this should always be zero
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
		return fmt.Errorf("not a maria address")
	}

	return nil
}

// returns true if CPU is to be halted and true if DLL has requested an interrupt
func (mar *Maria) Tick() (halt bool, nmi bool) {
	var dli bool

	// error is reset and will be set again as appropriate in this function
	mar.Error = nil

	// assuming ntsc for now
	mar.Coords.clk++
	if mar.Coords.clk > clksScanline {
		mar.Coords.clk = 0
		mar.Coords.scanline++
		mar.wsync = false

		if mar.Coords.scanline > ntscAbsoluteBottom {
			mar.Coords.scanline = 0
			mar.Coords.frame++

			// send current frame to renderer
			select {
			case mar.rendering <- mar.currentFrame:
			default:
			}

			// it's no longer safe to use that frame in this context. create a
			// new image to use for current frame
			//
			// this can almost certainly be improved in efficiency
			mar.currentFrame = mar.newImage()

		} else if mar.Coords.scanline == ntscVisibleTop {
			// enable DMA at start of visible screen
			mar.mstat = 0x00

			// start DLL reads
			_, err := mar.nextDLL(true)
			if err != nil {
				mar.Error = err
			}

		} else if mar.Coords.scanline > ntscVisibleBottom {
			mar.mstat = 0x80

		} else {
			var err error
			dli, err = mar.nextDLL(false)
			if err != nil {
				mar.Error = err
			}
		}

		// draw entire scanline in one go if DMA is enabled after the scanline increase
		if mar.mstat == 0x00 {
			// set entire scanline to background colour. individual pixels will
			// be changed according to the display lists
			sl := mar.Coords.scanline - ntscVisibleTop
			for clk := range clksVisible {
				mar.currentFrame.Set(clk, sl, mar.rgba[mar.bg])
			}

			switch mar.ctrl.dma {
			case 0x00:
				// treat as DMA being off but record a warning
				mar.Error = fmt.Errorf("%w: dma value of 0x00 in ctrl register is undefined", WarningErr)
			case 0x01:
				// treat as DMA being off but record a warning
				mar.Error = fmt.Errorf("%w: dma value of 0x01 in ctrl register is undefined", WarningErr)
			case 0x02:
				mar.nextDL(true)
				for !mar.DL.isEnd {
					switch mar.ctrl.readMode {
					case 0:
						for w := range mar.DL.width {
							a := ((uint16(mar.DL.highAddress) << 8) | uint16(mar.DL.lowAddress))
							if mar.DL.indirect {
								a += (uint16(mar.charbase) << 8)
							}
							a += uint16(mar.DLL.workingOffset) << 8
							a += uint16(w)

							b, err := mar.mem.Read(a)
							if err != nil {
								mar.Error = fmt.Errorf("%w: failed to read graphics byte", err)
							}

							if mar.DLL.h16 && (a&0x9000 == 0x9000) {
								continue // for width loop
							} else if mar.DLL.h8 && (a&0x8800 == 0x8800) {
								continue // for width loop
							}

							if mar.DL.writemode {
								// 160B
								for i := range 2 {
									c := (b >> (((1 - i) * 2) + 4)) & 0x03
									pi := (mar.DL.palette & 0x40) + ((b >> ((1 - i) * 2)) & 0x03)
									p := mar.palette[pi]
									if c > 0 {
										mar.currentFrame.Set(int(mar.DL.horizontalPosition)+(int(w)*2)+i, sl, mar.rgba[p[c-1]])
									} else if mar.ctrl.kanagroo {
										// mar.currentFrame.Set(int(mar.DL.horizontalPosition)+(int(w)*2)+i, sl, mar.rgba[mar.bg])
									}
								}
							} else {
								// 160A
								p := mar.palette[mar.DL.palette]
								for i := range 4 {
									c := (b >> ((3 - i) * 2)) & 0x03
									if c > 0 {
										mar.currentFrame.Set(int(mar.DL.horizontalPosition)+(int(w)*4)+i, sl, mar.rgba[p[c-1]])
									} else if mar.ctrl.kanagroo {
										// mar.currentFrame.Set(int(mar.DL.horizontalPosition)+(int(w)*4)+i, sl, mar.rgba[mar.bg])
									}
								}
							}
						}

					case 1:
						mar.Error = fmt.Errorf("%w: readmode value of 0x01 in ctrl register is undefined", WarningErr)
					case 2:
						mar.Error = fmt.Errorf("%w: readmode value of 0x02 in ctrl register is not fully emulated", WarningErr)
					case 3:
						mar.Error = fmt.Errorf("%w: readmode value of 0x03 in ctrl register is not fully emulated", WarningErr)
					}

					mar.nextDL(false)
				}
			case 0x03:
				// dma is off. showing only background colour
			}
		}
	}

	// return HALT signal if either WSYNC or DMA signal is enabled
	return mar.wsync || (mar.mstat == 0x00 && mar.Coords.clk > clksHBLANK), dli
}
