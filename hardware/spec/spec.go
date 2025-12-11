package spec

import (
	_ "embed"
	"image/color"
)

// NTSC palettes downloaded from
// https://forums.atariage.com/topic/210082-colorswhat-do-you-want/#comment-2716653
// using files in option (A)
//
// palettes/NTSC_A7800_CRTTV.pal
// palettes/PAL_A7800_CRTTV.pal

// preferring BRT palettes in trebor sub-directory

//go:embed "palettes/trebor/NTSC_A78_CRTTV_BRT.pal"
var ntscRaw []byte

//go:embed "palettes/trebor/PAL_A78_CRTTV_BRT.pal"
var palRaw []byte

const (
	ClksScanline    = 454
	ClksHBLANK      = 134
	ClksVisible     = 320
	ClksColourBurst = 16
)

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
		//
		// the number of visible lines (ie. the number of lines between VisibleTop and
		// VisibleBottom) cannot be more than 264. if it is then there are ROMs which may use
		// unitialised memory for the DLL data and trigger an interrupt. the best example of this is
		// the high score entry screen for Centipede (when the HSC is attached)
		ID:             "NTSC",
		VisibleTop:     16,
		VisibleBottom:  258,
		AbsoluteBottom: 263,
		SafeTop:        27,
		SafeBottom:     253,
		HorizScan:      15734.26,
	}

	PAL = Spec{
		// 	"For PAL consoles, there are a total of 313 rasters per frame. (~1/50th
		// 	per second). The 'visible' screen starts on raster 16 and ends on raster 308"
		ID:             "PAL",
		VisibleTop:     16,
		VisibleBottom:  308,
		SafeTop:        27,
		SafeBottom:     303,
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
