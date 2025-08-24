//go:build !wasm
// +build !wasm

package ebiten

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/jetsetilly/test7800/resources"
)

func onWindowOpen() (windowGeometry, error) {
	s, err := resources.Read("window")
	if err != nil {
		return windowGeometry{}, err
	}

	var g windowGeometry

	_, err = fmt.Sscanf(s, "%d %d %d %d", &g.x, &g.y, &g.w, &g.h)
	if err != nil {
		return windowGeometry{}, err
	}

	if !g.valid() {
		return g, nil
	}

	ebiten.SetWindowPosition(g.x, g.y)
	ebiten.SetWindowSize(g.w, g.h)

	return g, nil
}

func onWindowClose(g windowGeometry) error {
	if !g.valid() {
		return nil
	}

	s := fmt.Sprintf("%d %d %d %d", g.x, g.y, g.w, g.h)
	return resources.Write("window", s)
}
