package ebiten

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/jetsetilly/test7800/resources"
)

func onWindowOpen() error {
	s, err := resources.Read("window")
	if err != nil {
		return err
	}

	var x, y, w, h int

	_, err = fmt.Sscanf(s, "%d %d %d %d", &x, &y, &w, &h)
	if err != nil {
		return err
	}

	ebiten.SetWindowPosition(x, y)
	ebiten.SetWindowSize(w, h)

	return nil
}

func onCloseWindow() error {
	x, y := ebiten.WindowPosition()
	w, h := ebiten.WindowSize()
	s := fmt.Sprintf("%d %d %d %d", x, y, w, h)
	return resources.Write("window", s)
}
