package maria

import (
	"github.com/jetsetilly/test7800/hardware/clocks"
	"github.com/jetsetilly/test7800/hardware/spec"
)

const (
	// * the '7800 Software Guide' states in appendix 3 that, "DMA does not begin until 7 CPU (1.79 MHz)
	// cycles into each scan line"
	preDMA = (7 * clocks.MariaCycles)

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

	// the last header in the display list has a cost, even though it isn't
	// fully decoded
	dmaLastDLHeader = dmaShortDLHeader

	// "The end of VBLANK is made up of a DMA startup plus a Long shutdown."
	dmaEndofVBLANK = dmaStartLastInZone

	// additional DMA overhead in the event of an interrupt being triggered is
	// not mentioned in the '7800 Software Guide'. however both js7800 and mame
	// use a value of 17.
	//
	// I am using that value here because it improves the DMA timing as measured
	// by '7800 Test (NTSC) (20140406) (EF65C77A).a78' and also fixes a
	// rendering error in Xevious (a yellow line between the score and playfield
	// areas)
	dmaInterruptOverhead = 17

	// the maximum number of cycles available in DMA before the HSYNC
	dmaMaxCycles = spec.ClksScanline - preDMA
)
