//go:build wasm
// +build wasm

package ebiten

func onWindowOpen() (windowGeometry, error) {
	return windowGeometry{}, nil
}

func onWindowClose(g windowGeometry) error {
	return nil
}
