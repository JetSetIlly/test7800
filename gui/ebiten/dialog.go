package ebiten

import (
	"errors"
	"path/filepath"

	"github.com/jetsetilly/dialog"
)

func fileRequest(lastSelectedROM string) (string, error) {
	dlg := dialog.File()
	dlg = dlg.Title("Select 7800 ROM")
	dlg = dlg.Filter("7800 Files", "a78", "bin", "elf", "boot")
	dlg = dlg.Filter("A78 Files Only", "a78")
	dlg = dlg.Filter("All Files")
	dlg = dlg.SetStartDir(filepath.Dir(lastSelectedROM))
	filename, err := dlg.Load()
	if err != nil {
		if errors.Is(err, dialog.ErrCancelled) {
			return "", nil
		}
		return "", err
	}
	return filename, nil
}

func showError(msg string) {
	dialog.Message(msg).Error()
}
