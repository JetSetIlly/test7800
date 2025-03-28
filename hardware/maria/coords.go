package maria

import (
	"fmt"
)

type coords struct {
	Frame    int
	Scanline int
	Clk      int
}

func (c *coords) String() string {
	return fmt.Sprintf("frame: %d, scanline: %d, clk: %d", c.Frame, c.Scanline, c.Clk)
}

func (c *coords) ShortString() string {
	return fmt.Sprintf("%d/%03d/%03d", c.Frame, c.Scanline, c.Clk)
}

func (c *coords) Reset() {
	c.Frame = 0
	c.Scanline = 0
	c.Clk = 0
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
