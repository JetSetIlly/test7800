package maria

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware/memory/external"
	"github.com/jetsetilly/test7800/hardware/spec"
)

// Context allows Maria to signal a break
type Context interface {
	Break(error)
	Spec() spec.Spec
	UseOverlay() bool
}

type limiter interface {
	Wait()
}

// the wrapping error for any errors passed to Context.Break()
var ContextError = errors.New("maria")

// the values taken by mstat to indicate VBLANK. vblank is the only bit stored
// in mstat so a simple equate is sufficient
const (
	vblankEnable  = 0x80
	vblankDisable = 0x00
)

type frame struct {
	debug   bool
	top     int
	bottom  int
	left    int
	right   int
	main    *image.RGBA
	overlay *image.RGBA
}

type Maria struct {
	ctx Context
	g   *gui.GUI

	// frame limiter
	limit limiter

	bg       uint8
	wsync    bool
	palette  [8][3]uint8
	dpph     uint8
	dppl     uint8
	charbase uint8
	offset   uint8
	mstat    uint8 // bit 7 is true if VBLANK is enabled
	ctrl     mariaCtrl

	// if the colour burst signal has been sent to the TV. it will not be sent if the colour kill
	// bit was set in the ctrl register at beginning of the scanline when the colour burst is to be
	// sent
	colourBurst bool

	// lineram is where DL/DLL information is written to before being read and
	// rendered to the current frame
	lineram lineram

	// interface to console memory and address/data bus
	mem Memory

	// the current coordinates of the TV image
	Coords coords

	// the current television specificaion (NTSC, PAL, etc.)
	Spec spec.Spec

	// the image that is sent to the user interface
	currentFrame frame
	prevFrame    frame

	// interface to CPU (for debugging purposes only)
	cpu CPU

	// current DLL
	DLL dll
	DL  dl

	// the most recent DLLs. reset on start of DMA of a new frame. used for
	// debugging feedback
	RecentDLL []dll
	RecentDL  []dl

	// whether DMA is active at the current moment. it is enabled if ctrl.dma is
	// enabled when the clock counter reaches preDMA; and then disabled when the
	// number of required DMA cycles for the DLL is reacehd
	//
	// we also need to take into account the dmaLatch. this is set when the CPU is at the end of a
	// cycle. DMA will not start unless the latch is set
	dma bool

	// the clock on which the DMA actually started. the start clock can change depending on when the
	// dmaLatch is set
	dmaStart int

	// whether the dma has been active this scanline. this can be true even if the dma field is true
	// which can happen once DMA has ended. dmaLatched is reset at the start of a new scanline
	dmaLatched bool

	// the number of DMA cycles required to construct the scanline
	requiredDMACycles int

	// number of cycles before DMI is triggered
	interruptDelay int

	// the DLI signal is sent at the end of DMA but because we process the entirity
	// of the scanline as soon as DMA starts we store the signal until DMA has actually
	// finished
	dli bool
}

type Memory interface {
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
	ExternalDevice() *external.Device
}

type CPU interface {
	InInterrupt() bool
}

func Create(ctx Context, g *gui.GUI, mem Memory, cpu CPU, limit limiter) *Maria {
	mar := &Maria{
		ctx:   ctx,
		g:     g,
		mem:   mem,
		cpu:   cpu,
		Spec:  ctx.Spec(),
		limit: limit,
	}

	mar.lineram.initialise()
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

	mar.colourBurst = false
	mar.dma = false
	mar.dmaStart = 0
	mar.dmaLatched = false
	mar.requiredDMACycles = 0
	mar.interruptDelay = 0
	mar.dli = false

	mar.DL = dl{}
	mar.DLL = dll{}
	mar.RecentDL = mar.RecentDL[:0]
	mar.RecentDLL = mar.RecentDLL[:0]

	mar.mem.ExternalDevice().HLT(false)
}

func (mar *Maria) Label() string {
	return "MARIA"
}

func (mar *Maria) Status() string {
	if mar.wsync {
		return "WSYNC"
	}
	return mar.String()
}

func (mar *Maria) String() string {
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
	// From the '7800 Software Guide'
	// "All the palette and BACKGRND registers are write-only."

	switch idx {
	case 0x020:
		// background
	case 0x021:
		// palette[0][0]
	case 0x022:
		// palette[0][1]
	case 0x023:
		// palette[0][2]
	case 0x024:
		// wsync (write only)
	case 0x025:
		// palette[1][0]
	case 0x026:
		// palette[1][1]
	case 0x027:
		// palette[1][2]
	case 0x028:
		// maria status
		return mar.mstat, nil
	case 0x029:
		// palette[2][0]
	case 0x02a:
		// palette[2][1]
	case 0x02b:
		// palette[2][2]
	case 0x02c:
		// display list point high (write only)
	case 0x02d:
		// palette[3][0]
	case 0x02e:
		// palette[3][1]
	case 0x02f:
		// palette[3][2]
	case 0x030:
		// display list point low (write only)
	case 0x031:
		// palette[4][0]
	case 0x032:
		// palette[4][1]
	case 0x033:
		// palette[4][2]
	case 0x034:
		// character base address (write only)
	case 0x035:
		// palette[5][0]
	case 0x036:
		// palette[5][1]
	case 0x037:
		// palette[5][2]
	case 0x038:
		// reserved for future expansion. this should always be zero
		return mar.offset, nil
	case 0x039:
		// palette[6][0]
	case 0x03a:
		// palette[6][1]
	case 0x03b:
		// palette[6][2]
	case 0x03c:
		// maria control (write only)
		return 0, nil
	case 0x03d:
		// palette[7][0]
	case 0x03e:
		// palette[7][1]
	case 0x03f:
		// palette[7][2]
	default:
		return 0, fmt.Errorf("not a maria address (%#04x)", idx)
	}
	return 0, nil
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

func (mar *Maria) newFrame() {
	mar.prevFrame = mar.currentFrame

	mar.currentFrame.debug = mar.ctx.UseOverlay()
	if mar.currentFrame.debug {
		mar.currentFrame.left = 0
		mar.currentFrame.right = spec.ClksScanline
		mar.currentFrame.top = 0
		mar.currentFrame.bottom = mar.Spec.AbsoluteBottom
	} else {
		mar.currentFrame.left = spec.ClksHBLANK
		mar.currentFrame.right = spec.ClksScanline
		mar.currentFrame.top = mar.Spec.VisibleTop + 10
		mar.currentFrame.bottom = mar.Spec.VisibleBottom - 8
	}

	mar.currentFrame.main = image.NewRGBA(image.Rect(0, 0,
		mar.currentFrame.right-mar.currentFrame.left,
		mar.currentFrame.bottom-mar.currentFrame.top),
	)

	mar.currentFrame.overlay = image.NewRGBA(image.Rect(0, 0,
		mar.currentFrame.right-mar.currentFrame.left,
		mar.currentFrame.bottom-mar.currentFrame.top),
	)
}

func (mar *Maria) PushRender() {
	var cursor = [2]int{
		mar.Coords.Clk - mar.currentFrame.left,
		mar.Coords.Scanline - mar.currentFrame.top,
	}

	// send current frame to renderer
	select {
	case mar.g.SetImage <- gui.Image{
		Main:    mar.currentFrame.main,
		Overlay: mar.currentFrame.overlay,
		Prev:    mar.prevFrame.main,
		ID:      mar.Coords.ShortString(),
		Cursor:  cursor,
	}:
	default:
	}
}

// dmaLatch indicates that the DMA can begin if the preDMA has been reached.
//
// From the '7800 Software Guide'
// "The DMA start-up may be delayed if the 6502 clock isn't at the end of a cycle when DMA begins."
func (mar *Maria) Tick(dmaLatch bool) (dma bool, rdy bool, nmi bool) {
	mar.Coords.Clk++
	if mar.Coords.Clk >= spec.ClksScanline {
		mar.Coords.Clk = 0
		mar.Coords.Scanline++
		mar.wsync = false
		mar.dma = false
		mar.dmaLatched = false
		mar.requiredDMACycles = 0

		mar.mem.ExternalDevice().HLT(false)

		mar.lineram.newScanline()
		mar.RecentDL = mar.RecentDL[:0]

		if mar.Coords.Scanline >= mar.Spec.AbsoluteBottom {
			mar.Coords.Scanline = 0
			mar.Coords.Frame++

			mar.limit.Wait()
			mar.PushRender()

			// it's no longer safe to use that frame in this context. create a
			// new image to use for current frame
			//
			// this can almost certainly be improved in efficiency
			mar.newFrame()

		} else if mar.Coords.Scanline == mar.Spec.VisibleTop {
			mar.mstat = vblankDisable

			// "The end of VBLANK is made up of a DMA startup plus a Long shutdown."
			mar.requiredDMACycles += dmaEndofVBLANK

			// reset list of DLLs seen this frame
			mar.RecentDLL = mar.RecentDLL[:0]

		} else if mar.Coords.Scanline == mar.Spec.VisibleBottom {
			mar.mstat = vblankEnable
		}
	}

	// sent colour burst only if the colour kill bit in the ctrl register is cleared
	if mar.Coords.Clk == spec.ClksColourBurst {
		mar.colourBurst = !mar.ctrl.colourKill
	}

	// the x and y values are the frame coordinates where lineram information
	// (and debugging overlay information) is plotted. they are adjusted according to
	// whether the overlay is active or not
	x := mar.Coords.Clk - mar.currentFrame.left
	y := mar.Coords.Scanline - mar.currentFrame.top

	// reduce colour to greyscale if colourburst signal was not sent. an example of a game that uses
	// the colour kill bit to affect the colour burst signal is Midnight Mutants. the greyscale can
	// be seen in the text at the top of the screen. without correct handling of the colour kill bit
	// the text area will contain red pixels
	colourBurst := func(col color.RGBA) color.RGBA {
		if mar.colourBurst {
			return col
		}

		// if colour burst has not been sent for this scanline then the colour is reduced to grayscale
		Y := uint8(0.299*float64(col.R) + 0.587*float64(col.G) + 0.114*float64(col.B))
		return color.RGBA{R: Y, G: Y, B: Y, A: col.A}
	}

	// read from lineram and draw to screen on a clock-by-clock basis
	if mar.Coords.Scanline >= mar.currentFrame.top && mar.Coords.Scanline <= mar.currentFrame.bottom {
		if mar.Coords.Clk >= spec.ClksHBLANK && mar.Coords.Clk < spec.ClksScanline &&
			mar.Coords.Clk&0x01 == spec.ClksHBLANK&0x01 {

			e := mar.lineram.read(mar.Coords.Clk - spec.ClksHBLANK)
			if !e.set {
				mar.currentFrame.main.Set(x, y, colourBurst(mar.Spec.Palette[mar.bg]))
				mar.currentFrame.main.Set(x+1, y, colourBurst(mar.Spec.Palette[mar.bg]))
			} else {
				switch mar.ctrl.readMode {
				case 0:
					// 160A/B
					if e.idx == 0 {
						mar.currentFrame.main.Set(x, y, colourBurst(mar.Spec.Palette[mar.bg]))
						mar.currentFrame.main.Set(x+1, y, colourBurst(mar.Spec.Palette[mar.bg]))
					} else {
						mar.currentFrame.main.Set(x, y, colourBurst(mar.Spec.Palette[mar.palette[e.palette][e.idx-1]]))
						mar.currentFrame.main.Set(x+1, y, colourBurst(mar.Spec.Palette[mar.palette[e.palette][e.idx-1]]))
					}
				case 1:
					mar.ctx.Break(fmt.Errorf("%w: readmode value of 0x01 in ctrl register is undefined", ContextError))
				case 2:
					// 320B/D
					//
					// this readmode is different because some of the palette bits are used to supplement the
					// index value thereby forming a new index value. this means the values in the palette and
					// index fields of the linram entry are a little misleading
					//
					// the MAME method of constructing the data when writing into lineram perhaps makes more
					// sense, but it's only for modes 320B/D where this is an issue
					p := e.palette & 0x04
					d := e.idx & 0x02
					d |= (e.palette & 0x02) >> 1
					if d == 0 {
						mar.currentFrame.main.Set(x, y, colourBurst(mar.Spec.Palette[mar.bg]))
					} else {
						mar.currentFrame.main.Set(x, y, colourBurst(mar.Spec.Palette[mar.palette[p][d-1]]))
					}
					d = (e.idx & 0x01) << 1
					d |= e.palette & 0x01
					if d == 0 {
						mar.currentFrame.main.Set(x+1, y, colourBurst(mar.Spec.Palette[mar.bg]))
					} else {
						mar.currentFrame.main.Set(x+1, y, colourBurst(mar.Spec.Palette[mar.palette[p][d-1]]))
					}
				case 3:
					// 320A/C
					d := (e.idx >> 1) & 0x01
					if d == 0 {
						mar.currentFrame.main.Set(x, y, colourBurst(mar.Spec.Palette[mar.bg]))
					} else {
						mar.currentFrame.main.Set(x, y, colourBurst(mar.Spec.Palette[mar.palette[e.palette][1]]))
					}
					d = e.idx & 0x01
					if d == 0 {
						mar.currentFrame.main.Set(x+1, y, colourBurst(mar.Spec.Palette[mar.bg]))
					} else {
						mar.currentFrame.main.Set(x+1, y, colourBurst(mar.Spec.Palette[mar.palette[e.palette][1]]))
					}
				}
			}
		}
	}

	// scanline build is done at the start of DMA. note the careful use of mar.dmaLatched and
	// dmaLatch to ensure that this block is only executed once per scanline
	if !mar.dmaLatched && dmaLatch && mar.Coords.Clk >= preDMA {
		// DMA is only ever active when VBLANK is disabled
		if mar.mstat == vblankDisable {
			switch mar.ctrl.dma {
			case 0x00:
				mar.ctx.Break(fmt.Errorf("%w: dma value of 0x00 in ctrl register is undefined", ContextError))
			case 0x01:
				mar.ctx.Break(fmt.Errorf("%w: dma value of 0x01 in ctrl register is undefined", ContextError))
			case 0x02:
				// dma is now active
				mar.dma = true
				mar.dmaLatched = true
				mar.dmaStart = mar.Coords.Clk

				mar.mem.ExternalDevice().HLT(true)

				if mar.DLL.workingOffset == 0x00 {
					mar.requiredDMACycles += dmaStartLastInZone
				} else {
					mar.requiredDMACycles += dmaStart
				}

				err := mar.nextDL(true)
				if err != nil {
					mar.ctx.Break(fmt.Errorf("%w: %w", ContextError, err))
				}

				for !mar.DL.isEnd {
					// DMA cycle accumulation for DL header
					if mar.DL.long {
						mar.requiredDMACycles += dmaLongDLHeader
					} else {
						mar.requiredDMACycles += dmaShortDLHeader
					}

					// keeps track of whether the DMA cycle accumulation has happened for a holey
					// read already. we only want one dmaHoleyRead accumulation per display list
					var inHole bool

					for w := range mar.DL.width {
						if mar.requiredDMACycles > maxDMA {
							break // for loop
						}

						// write data to line ram
						write := func(b uint8, secondWrite bool) {
							dbl := mar.ctrl.charWidth && mar.DL.indirect

							// the offset is added to the base horizontal position of the DL
							offset := w
							if dbl {
								offset *= 2
								if secondWrite {
									offset++
								}
							}

							if mar.DL.writemode {
								for i := range 2 {
									c := (b >> (((1 - i) * 2) + 4)) & 0x03
									p := (mar.DL.palette & 0x04) + ((b >> ((1 - i) * 2)) & 0x03)
									x := int(mar.DL.horizontalPosition+(offset*2)+uint8(i)) * 2
									if x < spec.ClksVisible {
										if c > 0 || mar.ctrl.kangaroo {
											mar.lineram.write(x, p, c)
											mar.lineram.write(x+1, p, c)
										}
									}
								}
							} else {
								for i := range 4 {
									c := (b >> ((3 - i) * 2)) & 0x03
									x := int(mar.DL.horizontalPosition+(offset*4)+uint8(i)) * 2
									if x < spec.ClksVisible {
										if c > 0 || mar.ctrl.kangaroo {
											mar.lineram.write(x, mar.DL.palette, c)
											mar.lineram.write(x+1, mar.DL.palette, c)
										}
									}
								}
							}
						}

						// the basic address is the same for indirect and direct modes
						a := ((uint16(mar.DL.highAddress) << 8) | uint16(mar.DL.lowAddress))
						a += uint16(w)

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
								if !inHole {
									mar.requiredDMACycles += dmaHoleyRead
									inHole = true
								}
								continue // for width loop
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
								if !inHole {
									mar.requiredDMACycles += dmaHoleyRead
									inHole = true
								}
								continue // for width loop
							}

							b, err := mar.mem.Read(a)
							if err != nil {
								mar.ctx.Break(fmt.Errorf("%w: failed to read graphics byte (%w)", ContextError, err))
							}
							write(b, false)

							mar.requiredDMACycles += dmaDirectGfx
						}
					}

					mar.RecentDL = append(mar.RecentDL, mar.DL)
					err := mar.nextDL(false)
					if err != nil {
						mar.ctx.Break(fmt.Errorf("%w: %w", ContextError, err))
					}
				}

				// DLL sequence is reset at beginning of vblankDisable (ie. when scanline is
				// equal to 'visible top')
				ok, err := mar.nextDLL(mar.Coords.Scanline == mar.Spec.VisibleTop)
				if err != nil {
					mar.ctx.Break(fmt.Errorf("%w: %w", ContextError, err))
				}
				if ok {
					mar.RecentDLL = append(mar.RecentDLL, mar.DLL)

					// trigger DLI if necessary: "One of the bits of a DLL entry tells MARIA to
					// generate a Display List Interrupt (DLI) for that zone. The interrupt will
					// actually occur following DMA on the last line of the PREVIOUS zone."
					//
					// and from Appendix 3: "Another timing consideration is there is one MPU
					// (7.16 MHz) cycle between DMA shutdown and generation of a DLI."
					mar.dli = mar.DLL.dli

					// the interrupt will be sent when dma has finished
					if mar.dli {
						// additional DMA overhead in the event of an interrupt being triggered is
						// not mentioned in the '7800 Software Guide'. however both js7800 and mame
						// use a value of 17
						//
						// however I find that a value of 24 is required for Karetka to render the
						// bottom of the red play area correctly. the difference may be because of a
						// timing issue elsewhere in the emulation however
						//
						// a value of 24 is also good for Scrapyard Dog, the other game that seems
						// sensitive to this
						const dmaInterruptOverhead = 24
						mar.interruptDelay = dmaInterruptOverhead
					}
				}
			case 0x03:
				// dma is off. showing only background colour
			}
		}
	}

	// disable dma when the number of required cycles has passed
	if mar.Coords.Clk == mar.dmaStart+mar.requiredDMACycles {
		mar.dma = false
		mar.mem.ExternalDevice().HLT(false)
	}

	// plot debugging information
	if mar.currentFrame.debug {
		if mar.Coords.Clk > mar.currentFrame.left {
			if mar.dma {
				// create a striped effect to the DLL overlay by using a slightly different colour
				// value for every odd numbered DLL
				var v uint8
				if mar.DLL.ct&0x01 == 0x01 {
					v = 175
				} else {
					v = 255
				}

				// while DMA is active the debugging overlay is red
				//
				// there's a small flaw in this caused by the skew between how the screen is drawn
				// and where the DMA starts and maximum extent of DMA. see dma.go for more
				// discussion but basically, we draw the screen so that the HBLANK area is drawn
				// entirely on the left hand side. for DMA however, it's more correct to think of
				// DMA has being partly on the right and partly on the left
				//
				// rather than complicate the code however, we'll just live with the DMA indicator
				// being cropped
				mar.currentFrame.overlay.Set(x, y, color.RGBA{R: v, A: 255})
			} else if mar.wsync {
				// wsync overlay is blue
				mar.currentFrame.overlay.Set(x, y, color.RGBA{B: 255, A: 255})
			} else if mar.cpu.InInterrupt() {
				// debugging overlay is green for the duration the CPU is
				// executing instruction inside an interrupt
				mar.currentFrame.overlay.Set(x, y, color.RGBA{G: 255, A: 255})
			}

			// vblank is indicated by grey stripes
			if mar.mstat == vblankEnable && mar.Coords.Clk&0x07 == mar.Coords.Scanline&0x07 {
				mar.currentFrame.overlay.Set(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 100})
			}
		}
	}

	// dli signal is sent once DMA and delay has concluded
	var dli bool
	if !mar.dma {
		if mar.dli {
			mar.interruptDelay--
			if mar.interruptDelay == 0 {
				dli = true
				mar.dli = false
			}
		}
	}

	return mar.dma, !mar.wsync, dli
}
