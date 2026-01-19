//go:build wasm

package ebiten

func fileRequest(lastSelectedROM string) (string, error) {
	return lastSelectedROM, nil
}

func showError(msg string) {
}
