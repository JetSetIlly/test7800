package ebiten

import (
	_ "embed"

	"bytes"
	"image/color"
	"log"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

//go:embed "Hack-Regular.ttf"
var fontHack []byte

type ui struct {
	*ebitenui.UI
	baseFont text.Face
}

func createUI() *ui {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fontHack))
	if err != nil {
		log.Fatal(err)
	}

	var baseFont text.Face = &text.GoTextFace{
		Source: s,
		Size:   15,
	}

	ui := &ui{
		UI: &ebitenui.UI{
			Container: widget.NewContainer(),
		},
		baseFont: baseFont,
	}

	txt := widget.NewText(
		widget.TextOpts.Text("", &baseFont, color.White),
	)
	ui.Container.AddChild(txt)

	return ui
}
