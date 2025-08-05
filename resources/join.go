package resources

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jetsetilly/test7800/resources/fs"
)

// JoinPath prepends the supplied path with a with OS/build specific base
// paths, if required.
//
// The function creates all folders necessary to reach the end of sub-path. It
// does not otherwise touch or create the file.
func JoinPath(path ...string) (string, error) {
	// join supplied path
	p := filepath.Join(path...)

	var b string

	// resources are either in the portable path or the path returned by resourcePath(). the
	// resourcePath() function depends on how the program has been compiled - as a release binary or
	// as a development binary
	if checkPortable() {
		b = portablePath
	} else {
		var err error
		b, err = resourcePath()
		if err != nil {
			return "", err
		}
	}

	// do not prepend base path if it is already present
	if !strings.HasPrefix(p, b) {
		p = filepath.Join(b, filepath.Join(path...))
	}

	// check if path already exists
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}

	// create path if necessary
	if err := fs.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return "", err
	}

	return p, nil
}
