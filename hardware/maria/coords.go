package maria

import (
	"fmt"
)

type coords struct {
	frame    int
	scanline int
	clk      int
}

func (c *coords) String() string {
	return fmt.Sprintf("frame: %d, scanline: %d, clk: %d", c.frame, c.scanline, c.clk)
}

func (c *coords) ShortString() string {
	return fmt.Sprintf("%d/%03d/%03d", c.frame, c.scanline, c.clk)
}

func (c *coords) Reset() {
	c.frame = 0
	c.scanline = 0
	c.clk = 0
}

const (
	ntscHorizScan  = 15734.26
	palHorizScan   = 15625.00
	pal60HorizScan = 15625.00
	palMHorizScan  = 15734.26
	secamHorizScan = 15625.00
)

const clksScanline = 227
const clksHBLANK = 67
const clksVisible = 160

const (
	// "For NTSC consoles, there are a total of 263 rasters per frame (~1/60th
	// second). The 'visible' screen (during which MARIA attempts display)
	// starts on raster 16 and ends on raster 258."
	ntscVisibleTop     = 16
	ntscVisibleBottom  = 258
	ntscAbsoluteBottom = 263

	// 	"For PAL consoles, there are a total of 313 rasters per frame. (~1/50th
	// 	per second). The 'visible' screen starts on raster 16 and ends on raster
	// 	308"
	palVisibleTop     = 16
	palVisibleBottom  = 308
	palAbsoluteBottom = 313
)
