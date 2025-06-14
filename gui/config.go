package gui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

func onWindowOpen() error {
	pth, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	pth = filepath.Join(pth, "test7800", "window")
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
	pth, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	pth = filepath.Join(pth, "test7800")
	err = os.MkdirAll(pth, 0700)
	if err != nil {
		return err
	}

	pth = filepath.Join(pth, "window")
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
