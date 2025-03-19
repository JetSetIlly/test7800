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

// pre-processed RGB information
var ntscPalette [256]color.RGBA
var palPalette [256]color.RGBA

func init() {
	if len(ntscRaw) != 768 {
		panic("ntsc palette data is incorrect length. should be 768bytes (256bytes * 3)")
	}
	if len(palRaw) != 768 {
		panic("pal palette data is incorrect length. should be 768bytes (256bytes * 3)")
	}

	for i := range 256 {
		p := i * 3
		ntscPalette[i] = color.RGBA{R: ntscRaw[p], G: ntscRaw[p+1], B: ntscRaw[p+2]}
		palPalette[i] = color.RGBA{R: palRaw[p], G: palRaw[p+1], B: palRaw[p+2]}
	}
}
