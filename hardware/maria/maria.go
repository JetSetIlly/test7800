package maria

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/jetsetilly/test7800/hardware/clocks"
	"github.com/jetsetilly/test7800/ui"
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
	ctrl.dma = 3
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
	Spec() string
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
	ui  *ui.UI

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

	// lineram is implemented as a single line image. this isn't an exact
	// representation of lineram but it's workable and produces adequate results
	// for now
	lineram *image.RGBA

	// the current spec (NTSC, PAL, etc.)
	spec spec

	// frame limiter
	limiter *time.Ticker
}

type Memory interface {
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
}

func Create(ctx Context, ui *ui.UI, mem Memory) *Maria {
	mar := &Maria{
		ctx: ctx,
		ui:  ui,
		mem: mem,
	}

	switch ctx.Spec() {
	case "NTSC":
		mar.spec = ntsc
	case "PAL":
		mar.spec = pal
	default:
		panic("currently unsupported specification")
	}

	// calculate refresh rate and start frame limiter
	hz := mar.spec.horizScan / float64(mar.spec.absoluteBottom)

	// increase in refresh rate so that it better syncs with the audio. this is
	// definitely not ideal but it's okay for now
	// TODO: better interaction between frame limiter and audio
	mar.limiter = time.NewTicker(time.Second / time.Duration(hz))

	// allocate images representing lineram and the frame to be displayed
	mar.lineram = image.NewRGBA(image.Rect(0, 0, clksVisible, 1))
	mar.newFrame()

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

func (mar *Maria) Access(write bool, idx uint16, data uint8) (uint8, error) {
	if write {
		return data, mar.Write(idx, data)
	}
	return mar.Read(idx)
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

const (
	// Appendix 3: "DMA does not begin until 7 CPU (1.79 MHz) cycles into each scan line"
	preDMA = 7 * clocks.MariaCycles

	// from the table "DMA Timing"
	dmaStart           = 16
	dmaStartLastInZone = 24
	dmaDLHeader        = 8
	dmaLongDLHeader    = 10
	dmaDirectGfx       = 3
	dmaIndirectGfx     = 6
	dmaIndirectWideGfx = 9

	// "If holey DMA is enabled and graphics reads would reside in a DMA hole,
	// only 3 cycles of penalty for the graphic read is incurred, whatever the
	// sprite width is"
	dmaHoleyRead = 3

	// "The end of VBLANK is made up of a DMA startup plus a Long shutdown."
	dmaEndofVBLANK = dmaStartLastInZone

	// the maximum number of cycles available in DMA before the HSYNC
	dmaMaxCycles = clksScanline
)

func (mar *Maria) newFrame() {
	mar.currentFrame = image.NewRGBA(image.Rect(0, 0, clksVisible, mar.spec.visibleBottom-mar.spec.visibleTop))

	// make sure lineram is clear at beginning of new frame. probably not
	// necessary but not an expensive operation
	for clk := range clksVisible {
		mar.lineram.Set(clk, 0, color.Transparent)
	}
}

// returns true if CPU is to be halted and true if DLL has requested an interrupt
//
// note that the interrupt request will be returned at the beginning of DMA
// rather than at the end (which is when it would really occur). this doesn't
// matter however because the CPU will be stalled until the end of DMA
func (mar *Maria) Tick() (halt bool, interrupt bool) {
	// whether the current DLL indicates that an interrupt should occur
	var dli bool

	mar.Coords.Clk++
	if mar.Coords.Clk >= clksScanline {
		mar.Coords.Clk = 0
		mar.Coords.Scanline++
		mar.wsync = false
		mar.requiredDMACycles = 0

		if mar.Coords.Scanline >= mar.spec.absoluteBottom {
			mar.Coords.Scanline = 0
			mar.Coords.Frame++

			<-mar.limiter.C

			// send current frame to renderer
			select {
			case mar.ui.SetImage <- mar.currentFrame:
			default:
			}

			// it's no longer safe to use that frame in this context. create a
			// new image to use for current frame
			//
			// this can almost certainly be improved in efficiency
			mar.newFrame()

		} else if mar.Coords.Scanline == mar.spec.visibleTop-1 {
			mar.mstat = vblankDisable

			// "The end of VBLANK is made up of a DMA startup plus a Long shutdown."
			mar.requiredDMACycles += dmaEndofVBLANK

		} else if mar.Coords.Scanline == mar.spec.visibleBottom {
			mar.mstat = vblankEnable
		}
	}

	// scanline build is done at the start of DMA
	if mar.Coords.Clk == preDMA {
		// DMA is only ever active when VBLANK is disabled
		if mar.mstat == vblankDisable {

			// whether to trigger an interrupt at the end of the display list
			if mar.ctrl.dma != 3 {
				var err error
				dli, err = mar.nextDLL(mar.Coords.Scanline == mar.spec.visibleTop)
				if err != nil {
					mar.ctx.Break(fmt.Errorf("%w: %w", ContextError, err))
				}
			}

			// DMA cycle counting
			if mar.DLL.offset == 0 {
				mar.requiredDMACycles += dmaStartLastInZone
			} else {
				mar.requiredDMACycles += dmaStart
			}

			// the scanline value adjusted by where the top of the image is located
			sl := mar.Coords.Scanline - mar.spec.visibleTop

			for clk := range clksVisible {
				// set entire scanline to background colour
				mar.currentFrame.Set(clk, sl, mar.spec.palette[mar.bg])

				// copy line ram over background where the line ram is not transparent
				c := mar.lineram.RGBAAt(clk, 0)
				if c.A != 0 {
					mar.currentFrame.Set(clk, sl, c)
				}

				// clear the part of line ram we just copied
				mar.lineram.Set(clk, 0, color.Transparent)
			}

			switch mar.ctrl.dma {
			case 0x00:
				mar.ctx.Break(fmt.Errorf("%w: dma value of 0x00 in ctrl register is undefined", ContextError))
			case 0x01:
				mar.ctx.Break(fmt.Errorf("%w: dma value of 0x01 in ctrl register is undefined", ContextError))
			case 0x02:
				err := mar.nextDL(true)
				if err != nil {
					mar.ctx.Break(fmt.Errorf("%w: %w", ContextError, err))
				}
				for !mar.DL.isEnd {
					// the DMA can't go on too long so we exit early if appropriate
					if mar.requiredDMACycles > dmaMaxCycles {
						break // for loop
					}

					// DMA cycle accumulation for DL header
					if mar.DL.long {
						mar.requiredDMACycles += dmaLongDLHeader
					} else {
						mar.requiredDMACycles += dmaDLHeader
					}

					// the required DMA cycles value is adjusted once we know if the read will be
					// in holey memory

					// should there be a check here for execessive DMA cycles?
					//
					// if we add this check then then we need to take into account the possibility
					// of holey memory, which means we should also change the method of accumulation

					for w := range mar.DL.width {
						// the DMA can't go on too long so we exit early if appropriate
						if mar.requiredDMACycles > dmaMaxCycles {
							break // for loop
						}

						a := ((uint16(mar.DL.highAddress) << 8) | uint16(mar.DL.lowAddress))

						// width of the display list
						a += uint16(w)

						// write data to line ram
						write := func(b uint8, secondWrite bool) {
							dbl := mar.ctrl.charWidth && mar.DL.indirect

							pos := int(w)
							if dbl {
								pos *= 2
								if secondWrite {
									pos++
								}
							}

							switch mar.ctrl.readMode {
							case 0:
								if mar.DL.writemode {
									// 160B
									for i := range 2 {
										c := (b >> (((1 - i) * 2) + 4)) & 0x03
										pi := (mar.DL.palette & 0x40) + ((b >> ((1 - i) * 2)) & 0x03)
										p := mar.palette[pi]
										x := (int(mar.DL.horizontalPosition) + (pos * 2) + i) * 2
										if c > 0 {
											mar.lineram.Set(x, 0, mar.spec.palette[p[c-1]])
											mar.lineram.Set(x+1, 0, mar.spec.palette[p[c-1]])
										} else if mar.ctrl.kanagroo {
											mar.lineram.Set(x, 0, mar.spec.palette[mar.bg])
											mar.lineram.Set(x+1, 0, mar.spec.palette[p[c-1]])
										}
									}
								} else {
									// 160A
									p := mar.palette[mar.DL.palette]
									for i := range 4 {
										c := (b >> ((3 - i) * 2)) & 0x03
										x := (int(mar.DL.horizontalPosition) + (pos * 4) + i) * 2
										if c > 0 {
											mar.lineram.Set(x, 0, mar.spec.palette[p[c-1]])
											mar.lineram.Set(x+1, 0, mar.spec.palette[p[c-1]])
										} else if mar.ctrl.kanagroo {
											mar.lineram.Set(x, 0, mar.spec.palette[mar.bg])
											mar.lineram.Set(x+1, 0, mar.spec.palette[mar.bg])
										}
									}
								}
							case 1:
								mar.ctx.Break(fmt.Errorf("%w: readmode value of 0x01 in ctrl register is undefined", ContextError))
							case 2:
								if mar.DL.writemode {
									// 320B
								} else {
									// 320D
								}
							case 3:
								if mar.DL.writemode {
									// 320C
								} else {
									// 320A
									p := mar.palette[mar.DL.palette]
									for i := range 8 {
										// note that the horizontal position for 320 modes are doubled by the Maria
										// when writing to line ram. this gives an effective resolution of 160
										// pixels
										x := (int(mar.DL.horizontalPosition*2) + (int(pos) * 8) + i)
										if ((b << i) & 0x80) != 0 {
											mar.lineram.Set(x, 0, mar.spec.palette[p[1]])
										} else if mar.ctrl.kanagroo {
											mar.lineram.Set(x, 0, mar.spec.palette[mar.bg])
										}
									}
								}
							}
						}

						if mar.DL.indirect {
							b, err := mar.mem.Read(a)
							if err != nil {
								mar.ctx.Break(fmt.Errorf("%w: failed to read graphics byte (%w)", ContextError, err))
							}

							a = (uint16(mar.charbase) << 8) | uint16(b)

							// we'll be reading graphics data with this address so we add the working
							// offset to the high address byte (see comment above)
							a += uint16(mar.DLL.workingOffset) << 8

							// if this address is in a hole then all addresses in the DL will
							// be in the hole also
							if mar.DLL.inHole(a) {
								mar.requiredDMACycles += dmaHoleyRead
								break // for width loop
							}

							b, err = mar.mem.Read(a)
							if err != nil {
								mar.ctx.Break(fmt.Errorf("%w: failed to read graphics byte (%w)", ContextError, err))
							}
							write(b, false)

							if mar.ctrl.charWidth {
								b, err = mar.mem.Read(a + 1)
								if err != nil {
									mar.ctx.Break(fmt.Errorf("%w: failed to read graphics byte (%w)", ContextError, err))
								}
								write(b, true)

								mar.requiredDMACycles += dmaIndirectWideGfx
							} else {
								mar.requiredDMACycles += dmaIndirectGfx
							}

						} else {
							// "Each time graphics data is to be fetched OFFSET is added to the specified
							// High address byte, to determine the actual address where the data should
							// be found"
							a += uint16(mar.DLL.workingOffset) << 8

							// if this address is in a hole then all addresses in the DL will
							// be in the hole also
							if mar.DLL.inHole(a) {
								mar.requiredDMACycles += dmaHoleyRead
								break // for width loop
							}

							// DMA accumulation for direct gfx reads is simple
							mar.requiredDMACycles += dmaDirectGfx

							b, err := mar.mem.Read(a)
							if err != nil {
								mar.ctx.Break(fmt.Errorf("%w: failed to read graphics byte (%w)", ContextError, err))
							}
							write(b, false)
						}
					}

					err := mar.nextDL(false)
					if err != nil {
						mar.ctx.Break(fmt.Errorf("%w: %w", ContextError, err))
					}
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
