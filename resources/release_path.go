//go:build release
// +build release

package resources

import (
	"os"
	"path/filepath"
)

const configDir = "test7800"

func resourcePath() (string, error) {
	p, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(p, configDir), nil
}
