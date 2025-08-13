package maria

import (
	"github.com/jetsetilly/test7800/hardware/clocks"
	"github.com/jetsetilly/test7800/hardware/spec"
)

const (
	// the pre-DMA value seems to be the key to getting the DMA timing correct. the two games we're
	// using to help get the value correct is Summer Games and Karateka
	//
	// for the Summer Games ROM the line just below the judge's scores will flicker if this preDMA
	// value is not correct
	//
	// for Karateka the very bottom of the red of the game area will not extend all the way to the
	// right of the screen if the preDMA is not correct
	//
	// Count   Summer    Karateka
	//  7       N         N
	//  8       Y         N
	//  14      Y         N
	//  15      Y         Y
	//  16      Y         Y
	//  17      Y         Y
	//  18      N         Y
	//
	// a value of 15 is the lowest value for which Summer Games and Karateka displays correctly
	//
	// this value is not supported in any of the existing 7800 research documentation. appendix 3 of
	// '7800 Software Guide' statues that "DMA does not begin until 7 CPU (1.79 MHz) cycles into
	// each scan line"
	//
	// it also acknowledges that "there is some uncertainty as to the number of cycles DMA will
	// require, because the internal MARIA chip timing resolution is 7.16 MHz, while the 6502 runs
	// at either 1.79 MHz or 1.19MHz. As a result, it is not known how many extra cycles will be
	// needed in DMA startup/shutdown to make the 6502 happy"
	//
	// the fact that the document takes about the CPU running at varying speeds suggests that the
	// author is looking at this problem from a slightly different perspective. the varying CPU
	// speed is taken care of in the console.Step() function and the Maria is stepped according to
	// the current speed. I believe that this means that the number of cycles taken by the Maria is
	// constant. so rather than saying DMA takes (approx) 7 CPU cycles, we can say that it takes
	// exactly N cycles
	//
	// karateka
	// --------
	// a strong indicator that 15 is the correct value is how Karateka sets the black background
	// value at the end of the red play area (scanline 219). it deliberately uses two NOP
	// instructions to push back the write until the beginning of the next scanline
	//
	// the value of 15 is approximate but it is no more than 16. if it was 16 cycles there would be
	// no need for the second NOP. the absolute lowest value seems to be 14.25
	//
	// ballblazer
	// ----------
	// a value over 10 causes ballblazer to render background pixels in the middle of the playfield
	// and a value of less than 4 causes other issues
	//
	// crossbow
	// --------
	// a value of under 10 causes severe graphics corruption in crossbow
	//
	// the precise requirements of crossbow and ballblazer strongly suggest that the preDMA value
	// must be exactly 10. however, this leaves us with the problem of karateka
	//
	// karateka (again)
	// ----------------
	// an alternative strategy is required for karateka which isn't supported by anything in the
	// existing research material. for the case of background writes, it has been decided that the
	// write should be delayed by 19 cycles (see the Write() function in maria.go)
	//
	// this isn't a great solution and without more supporting evidince I'm hesitant to accept it as
	// the correct solution
	//
	preDMA = (10 * clocks.MariaCycles)

	// from the table "DMA Timing" in the '7800 Software Guide'
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
	dmaMaxCycles = spec.ClksScanline
)
