//go:build !release
// +build !release

package resources

const configDir = ".test7800"

func resourcePath() (string, error) {
	return configDir, nil
}
