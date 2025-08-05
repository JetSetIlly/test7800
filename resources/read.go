package resources

import (
	"fmt"
	"io"
	"os"
)

func Read(filename string) (string, error) {
	pth, err := JoinPath(filename)
	if err != nil {
		return "", err
	}

	f, err := os.Open(pth)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func Write(filename string, content string) error {
	pth, err := JoinPath(filename)
	if err != nil {
		return err
	}

	f, err := os.Create(pth)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := f.WriteString(content)
	if err != nil {
		return err
	}
	if n != len(content) {
		return fmt.Errorf("content not completely written")
	}

	return nil
}
