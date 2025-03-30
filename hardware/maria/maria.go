package maria

import (
	"errors"
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/jetsetilly/test7800/hardware/clocks"
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
	ctrl.border = data&0x08 == 0x08
	ctrl.kanagroo = data&0x04 == 0x04
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

// Context allows Maria to signal a break
type Context interface {
	Break(error)
}

// the wrapping error for any errors passed to Context.Break()
var ContextError = errors.New("maria")

// the values taken by mstat to indicate VBLANK. vblank is the only bit stored
// in mstat so a simple equate is sufficient
const (
	vblankEnable  = 0x80
	vblankDisable = 0x00
)

type Maria struct {
	ctx Context

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

	// the number of DMA cycles required to construct the scanline
	requiredDMACycles int

	// read-only registers
	mstat uint8 // bit 7 is true if VBLANK is enabled

	// interface to console memory
	mem Memory

	// the current coordinates of the TV image
	Coords coords

	// pixels for current frame
	currentFrame *image.RGBA
	rendering    chan *image.RGBA

	// the top of the image is not necessarily scanline zero
	imageTop int

	// the current spec (decided via the BIOS)
	spec spec

	// frame limiter
	limiter *time.Ticker

	// creating a new image depends on the tv specification. the newImage()
	// function returns an appropriately sized image for the specification
	newImage func() *image.RGBA
}

type Memory interface {
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
}

func Create(ctx Context, mem Memory, spec string, rendering chan *image.RGBA) *Maria {
	mar := &Maria{
		ctx:       ctx,
		mem:       mem,
		rendering: rendering,
	}

	switch strings.ToUpper(spec) {
	case "NTSC":
		mar.spec = ntsc
	case "PAL":
		mar.spec = pal
	default:
		panic("currently unsupported specification")
	}

	// calculate refresh rate and start frame limiter
	hz := mar.spec.horizScan / float64(mar.spec.absoluteBottom)
	mar.limiter = time.NewTicker(time.Second / time.Duration(hz))

	mar.imageTop = mar.spec.visibleTop

	mar.newImage = func() *image.RGBA {
		return image.NewRGBA(image.Rect(0, mar.imageTop, clksVisible, mar.spec.visibleBottom-mar.imageTop))
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
	mar.mstat = vblankDisable
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

// Appendix 3: "DMA does not begin until 7 CPU (1.79 MHz) cycles into each
// scan line"
const preDMA = 7 * clocks.MariaCycles

const (
	dmaStart           = 16
	dmaStartLastInZone = 24
	dmaDLHeader        = 8
	dmaLongDLHeader    = 10
	dmaDirectGfx       = 3
	dmaIndirectGfx     = 6
	dmaIndirectWideGfx = 9
)

// returns true if CPU is to be halted and true if DLL has requested an interrupt
//
// note that the interrupt request will be returned at the beginning of DMA
// rather than at the end (which is when it would really occur). this doesn't
// matter however because the CPU will be stalled until the end of DMA
func (mar *Maria) Tick() (halt bool, interrupt bool) {
	// whether the current DLL indicates that an interrupt should occur
	var dli bool

	mar.Coords.Clk++
	if mar.Coords.Clk > clksScanline {
		mar.Coords.Clk = 0
		mar.Coords.Scanline++
		mar.wsync = false

		if mar.Coords.Scanline > mar.spec.absoluteBottom {
			mar.Coords.Scanline = 0
			mar.Coords.Frame++

			<-mar.limiter.C

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

		} else if mar.Coords.Scanline == mar.spec.visibleTop {
			mar.mstat = vblankDisable
		} else if mar.Coords.Scanline == mar.spec.visibleBottom {
			mar.mstat = vblankEnable
		}
	}

	// scanline build is done at the start of DMA
	if mar.Coords.Clk == preDMA {
		// DMA is only ever active when VBLANK is disabled
		if mar.mstat == vblankDisable {

			// whether to trigger an interrupt at the end of the display list
			var err error
			dli, err = mar.nextDLL(mar.Coords.Scanline == mar.spec.visibleTop)
			if err != nil {
				mar.ctx.Break(fmt.Errorf("%w: %w", ContextError, err))
			}

			// DMA cycle counting
			if mar.DLL.offset == 0 {
				mar.requiredDMACycles = dmaStartLastInZone
			} else {
				mar.requiredDMACycles = dmaStart
			}

			// the scanline value adjusted by where the top of the image is located
			sl := mar.Coords.Scanline - mar.imageTop

			// set entire scanline to background colour. individual pixels will
			// be changed according to the display lists
			for clk := range clksVisible {
				mar.currentFrame.Set(clk, sl, mar.spec.palette[mar.bg])
			}

			switch mar.ctrl.dma {
			case 0x00:
				mar.ctx.Break(fmt.Errorf("%w: dma value of 0x00 in ctrl register is undefined", ContextError))
			case 0x01:
				mar.ctx.Break(fmt.Errorf("%w: dma value of 0x01 in ctrl register is undefined", ContextError))
			case 0x02:
				mar.nextDL(true)
				for !mar.DL.isEnd {
					// DMA cycle counting
					if mar.DL.long {
						mar.requiredDMACycles += dmaLongDLHeader
					} else {
						mar.requiredDMACycles += dmaDLHeader
					}
					if mar.DL.indirect {
						if mar.ctrl.charWidth {
							mar.requiredDMACycles += int(mar.DL.width) * dmaIndirectWideGfx
						} else {
							mar.requiredDMACycles += int(mar.DL.width) * dmaIndirectGfx
						}
					} else {
						mar.requiredDMACycles += int(mar.DL.width) * dmaDirectGfx
					}

					switch mar.ctrl.readMode {
					case 0:
						for w := range mar.DL.width {
							a := ((uint16(mar.DL.highAddress) << 8) | uint16(mar.DL.lowAddress))

							// width of the display list
							a += uint16(w)

							if !mar.DL.indirect {
								// "Each time graphics data is to be fetched OFFSET is added to the specified
								// High address byte, to determine the actual address where the data should
								// be found"
								a += uint16(mar.DLL.workingOffset) << 8

								if mar.DLL.inHole(a) {
									continue // for width loop
								}
							}

							b, err := mar.mem.Read(a)
							if err != nil {
								mar.ctx.Break(fmt.Errorf("%w: failed to read graphics byte (%w)", ContextError, err))
							}

							if mar.DL.indirect {
								a = (uint16(mar.charbase) << 8) | uint16(b)

								// we'll be reading graphics data with this address so we add the working
								// offset to the high address byte (see comment above)
								a += uint16(mar.DLL.workingOffset) << 8

								if mar.DLL.inHole(a) {
									continue // for width loop
								}

								b, err = mar.mem.Read(a)
								if err != nil {
									mar.ctx.Break(fmt.Errorf("%w: failed to read graphics byte (%w)", ContextError, err))
								}
							}

							if mar.DL.writemode {
								// 160B
								for i := range 2 {
									c := (b >> (((1 - i) * 2) + 4)) & 0x03
									pi := (mar.DL.palette & 0x40) + ((b >> ((1 - i) * 2)) & 0x03)
									p := mar.palette[pi]
									if c > 0 {
										mar.currentFrame.Set(int(mar.DL.horizontalPosition)+(int(w)*2)+i, sl, mar.spec.palette[p[c-1]])
									} else if mar.ctrl.kanagroo {
										mar.currentFrame.Set(int(mar.DL.horizontalPosition)+(int(w)*2)+i, sl, mar.spec.palette[mar.bg])
									}
								}
							} else {
								// 160A
								p := mar.palette[mar.DL.palette]
								for i := range 4 {
									c := (b >> ((3 - i) * 2)) & 0x03
									if c > 0 {
										mar.currentFrame.Set(int(mar.DL.horizontalPosition)+(int(w)*4)+i, sl, mar.spec.palette[p[c-1]])
									} else if mar.ctrl.kanagroo {
										mar.currentFrame.Set(int(mar.DL.horizontalPosition)+(int(w)*4)+i, sl, mar.spec.palette[mar.bg])
									}
								}
							}
						}

					case 1:
						mar.ctx.Break(fmt.Errorf("%w: readmode value of 0x01 in ctrl register is undefined", ContextError))
					case 2:
						mar.ctx.Break(fmt.Errorf("%w: readmode value of 0x01 in ctrl register is not fully emulated", ContextError))
					case 3:
						mar.ctx.Break(fmt.Errorf("%w: readmode value of 0x01 in ctrl register is not fully emulated", ContextError))
					}

					mar.nextDL(false)
				}
			case 0x03:
				// dma is off. showing only background colour
			}
		}
	}

	if mar.requiredDMACycles+preDMA > clksScanline {
		mar.ctx.Break(fmt.Errorf("%w: number of required DMA cycles is too much (%d)", ContextError, mar.requiredDMACycles))
	}

	// return HALT signal if either WSYNC or DMA is enabled
	return mar.wsync || (mar.mstat == vblankDisable &&
		mar.Coords.Clk >= preDMA &&
		mar.Coords.Clk <= (preDMA+mar.requiredDMACycles)), dli
}
