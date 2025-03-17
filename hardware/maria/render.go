package maria

import (
	_ "embed"
	"image"
	"image/color"
)

// palette downloaded from https://forums.atariage.com/topic/210082-colorswhat-do-you-want/#comment-2716653
// using files in option (A)

//go:embed "palettes/NTSC_A7800_CRTTV.pal"
var raw []byte

// pre-processed RGB information
var palette [256]color.RGBA

func init() {
	if len(raw) != 768 {
		panic("palette data is incorrect length. should be 768bytes (256bytes * 3)")
	}

	for i := range 256 {
		p := i * 3
		palette[i] = color.RGBA{R: raw[p], G: raw[p+1], B: raw[p+2]}
	}
}

// image is twice the width of clksVisible so we can support 320A, 320B, 320C and 320D
func newImage() *image.RGBA {
	return image.NewRGBA(image.Rect(0, 0, clksVisible*2, ntscVisibleBottom-ntscVisibleTop))
}
