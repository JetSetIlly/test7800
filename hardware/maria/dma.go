package maria

import (
	"github.com/jetsetilly/test7800/hardware/clocks"
	"github.com/jetsetilly/test7800/hardware/spec"
)

const (
	// the pre-DMA value seems to be the key to getting the DMA timing correct. there are four games
	// games that we have used we're using to help get the value correct. the games are Summer
	// Games, Karateka, Ballblazer and Crossbow. we'll concentrate on the first two games for now
	//
	// for the Summer Games ROM the line just below the judge's scores will flicker if this preDMA
	// value is not correct
	//
	// for Karateka the very bottom of the red of the game area will not extend all the way to the
	// right of the screen if the preDMA is not correct
	//
	// Factor    Summer    Karateka
	//  7          N         N
	//  8          Y         N
	//  14         Y         N
	//  15         Y         Y
	//  16         Y         Y
	//  17         Y         Y
	//  18         N         Y
	//
	// a value of 15 is the lowest value for which Summer Games and Karateka displays correctly
	//
	// the other two games suggest quite different values however. In the case of Ballblazer a value
	// over 10 causes red background pixels in the middle of the playfield; and a value of less than 4
	// causes other more severe issues
	//
	// for Crossbow, on the other hand, a value of under 10 causes severe graphics corruption in
	// crossbow.
	//
	// so taking these two games into account there is a very specific value of 10
	//
	//
	// the precise requirements of crossbow and ballblazer strongly suggest that the preDMA value
	// is exactly 10. (note that this contradicts the '7800 Software Guide' when it says that there is
	// "some uncertainty" with regard to this value)
	//
	// a value of 10 is also satisfactory for Summer Games. it does not solve the Karateka problem
	// however. for that problem we need to take into consideration the length of the pixel
	// pipeline. see the pipelineLength constant below
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

	// the number of pixels to read ahead in the lineram when decoding the fetch instructions. this
	// is a theoretical mechanism which I believe must exist in order to account for some DMA
	// related effects in some games - the Karateka background problem described in the comments for
	// the preDMA comment
	//
	// when reading the fetch instructions from lineram, is is postulated that the amount of time
	// required to decode the instruction and to reference the colour registers is not in absolute
	// lockstep with the TV raster scan. instead, we begin the decoding for pixel X of a scanline at
	// position (X-pipelineLength)
	pipelineLength = 18
)
