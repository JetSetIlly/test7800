package maria

import (
	_ "embed"
	"image/color"
)

// palettes downloaded from
// https://forums.atariage.com/topic/210082-colorswhat-do-you-want/#comment-2716653
// using files in option (A)

//go:embed "palettes/NTSC_A7800_CRTTV.pal"
var ntscRaw []byte

//go:embed "palettes/PAL_A7800_CRTTV.pal"
var palRaw []byte

const clksScanline = 454
const clksHBLANK = 134
const clksVisible = 320

type spec struct {
	palette        [256]color.RGBA
	visibleTop     int
	visibleBottom  int
	safeTop        int
	safeBottom     int
	absoluteBottom int
	horizScan      float64
}

var ntsc spec
var pal spec

func init() {
	ntsc = spec{
		// "For NTSC consoles, there are a total of 263 rasters per frame (~1/60th
		// second). The 'visible' screen (during which MARIA attempts display)
		// starts on raster 16 and ends on raster 258."
		visibleTop:     16,
		visibleBottom:  258,
		safeTop:        41,
		safeBottom:     233,
		absoluteBottom: 263,
		horizScan:      15734.26,
	}

	pal = spec{
		// 	"For PAL consoles, there are a total of 313 rasters per frame. (~1/50th
		// 	per second). The 'visible' screen starts on raster 16 and ends on raster
		// 	308"
		visibleTop:     16,
		visibleBottom:  308,
		safeTop:        41,
		safeBottom:     233,
		absoluteBottom: 313,
		horizScan:      15625.00,
	}

	if len(ntscRaw) != 768 {
		panic("ntsc palette data is incorrect length. should be 768bytes (256bytes * 3)")
	}
	if len(palRaw) != 768 {
		panic("pal palette data is incorrect length. should be 768bytes (256bytes * 3)")
	}

	for i := range 256 {
		p := i * 3
		ntsc.palette[i] = color.RGBA{R: ntscRaw[p], G: ntscRaw[p+1], B: ntscRaw[p+2]}
		pal.palette[i] = color.RGBA{R: palRaw[p], G: palRaw[p+1], B: palRaw[p+2]}
	}
}
