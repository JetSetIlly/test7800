package ebiten

import (
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/jetsetilly/test7800/resources"
)

func onWindowOpen() error {
	pth, err := resources.JoinPath("window")
	if err != nil {
		return err
	}

	f, err := os.Open(pth)
	if err != nil {
		return err
	}
	defer f.Close()

	var x, y, w, h int

	n, err := fmt.Fscanf(f, "%d %d %d %d", &x, &y, &w, &h)
	if err != nil {
		return err
	}
	if n != 4 {
		return fmt.Errorf("%s is malformed", pth)
	}

	ebiten.SetWindowPosition(x, y)
	ebiten.SetWindowSize(w, h)

	return nil
}

func onCloseWindow() error {
	pth, err := resources.JoinPath("window")
	if err != nil {
		return err
	}

	f, err := os.Create(pth)
	if err != nil {
		return err
	}
	defer f.Close()

	x, y := ebiten.WindowPosition()
	w, h := ebiten.WindowSize()
	fmt.Fprintf(f, "%d %d %d %d", x, y, w, h)

	return nil
}
