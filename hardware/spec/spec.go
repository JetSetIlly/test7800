package spec

import (
	_ "embed"
	"image/color"
)

// NTSC palette downloaded from
// https://forums.atariage.com/topic/210082-colorswhat-do-you-want/#comment-2716653
// using files in option (A)

//go:embed "palettes/trebor/NTSC_A78_CRTTV_BRT.pal"
var ntscRaw []byte

// PAL palette downloaded from
// https://forums.atariage.com/topic/383566-emulator-test7800/#comment-5700807

//go:embed "palettes/trebor/PAL_A78_CRTTV_BRT.pal"
var palRaw []byte

const ClksScanline = 454
const ClksHBLANK = 134
const ClksVisible = 320
const ClksColourBurst = 16

type Spec struct {
	ID             string
	Palette        [256]color.RGBA
	VisibleTop     int
	VisibleBottom  int
	SafeTop        int
	SafeBottom     int
	AbsoluteBottom int
	HorizScan      float64
}

var NTSC Spec
var PAL Spec

func init() {
	NTSC = Spec{
		// "For NTSC consoles, there are a total of 263 rasters per frame (~1/60th
		// second). The 'visible' screen (during which MARIA attempts display)
		// starts on raster 16 and ends on raster 258."
		ID:             "NTSC",
		VisibleTop:     16,
		VisibleBottom:  259,
		SafeTop:        41,
		SafeBottom:     233,
		AbsoluteBottom: 263,
		HorizScan:      15734.26,
	}

	PAL = Spec{
		// 	"For PAL consoles, there are a total of 313 rasters per frame. (~1/50th
		// 	per second). The 'visible' screen starts on raster 16 and ends on raster
		// 	308"
		ID:             "PAL",
		VisibleTop:     16,
		VisibleBottom:  309,
		SafeTop:        41,
		SafeBottom:     283,
		AbsoluteBottom: 313,
		HorizScan:      15625.00,
	}

	if len(ntscRaw) != 768 {
		panic("ntsc palette data is incorrect length. should be 768bytes (256bytes * 3)")
	}
	if len(palRaw) != 768 {
		panic("pal palette data is incorrect length. should be 768bytes (256bytes * 3)")
	}

	for i := range 256 {
		p := i * 3
		NTSC.Palette[i] = color.RGBA{R: ntscRaw[p], G: ntscRaw[p+1], B: ntscRaw[p+2], A: 255}
		PAL.Palette[i] = color.RGBA{R: palRaw[p], G: palRaw[p+1], B: palRaw[p+2], A: 255}
	}
}
