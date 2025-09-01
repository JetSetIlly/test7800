package maria

import (
	"github.com/jetsetilly/test7800/hardware/clocks"
	"github.com/jetsetilly/test7800/hardware/spec"
)

const (
	// the '7800 Software Guide' states in appendix 3 that, "DMA does not begin until 7 CPU (1.79 MHz)
	// cycles into each scan line"
	//
	// the reason for the difference of 7 and 10 is, I believe, due to differences in where the
	// scanline is considered to start. to be clear, I consider the start of the scanline to be when
	// HBLANK starts. I think it's possible that the research that informs the `7800 Software Guide`
	// considers the start of scanline to the end of HSYNC
	//
	// this impacts the dmaMaxCycles value calculated below. I'm not convinced the preDMA figure is
	// correct but it works and may suggest an error somewhere else in the emulation
	//
	// reasoning
	// ---------
	// the maximum preDMA value tolerated by Ballblazer is 10. if it is larger than 10 then the
	// background starts showing through as red pixels in the playfield. this happens because the
	// displaylist is changed too soon, and that causes the lineram to contain the wrong value for
	// when it is drawn to screen on the following scanline. the errant red pixels can be seen on
	// scanline 85 of the main play screen but the change of displaylist happens on scanline 84.
	preDMA = (10 * clocks.MariaCycles)

	// from the table "DMA Timing" in the '7800 Software Guide'
	dmaStart           = 16
	dmaStartLastInZone = 24
	dmaShortDLHeader   = 8
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
	//
	// the value of 435 has been determined to be the minimum required for Super Skateboarding to
	// render correctly. the score line of that game has many DL entries and needs as many cycles as
	// possible
	//
	// so starting with the number of maria cycles in the entire scanline we subtract the number of
	// cycles determined to be in the 'preDMA' phase and then adjust for the skew caused by the
	// difference in measurement
	//
	// what we're saying here is this: what we consider to be the end of the scanline and the point
	// when DMA is forced to end (if it hasn't already) is approximately 5 CPU cycles
	//
	// also note that the skew also effects the drawing of DMA extent for the DLL zones
	dmaMaxCycles = spec.ClksScanline - preDMA + (5 * clocks.MariaCycles) + 1
)
