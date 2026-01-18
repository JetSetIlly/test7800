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

func createUI() *ebitenui.UI {
	ui := &ebitenui.UI{
		Container: widget.NewContainer(),
	}
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fontHack))
	if err != nil {
		log.Fatal(err)
	}

	var fontFace text.Face = &text.GoTextFace{
		Source: s,
		Size:   15,
	}

	helloWorldLabel := widget.NewText(
		widget.TextOpts.Text("Test7800", &fontFace, color.White),
	)

	ui.Container.AddChild(helloWorldLabel)

	return ui
}
