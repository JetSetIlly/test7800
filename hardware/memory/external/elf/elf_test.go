package elf_test

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/jetsetilly/test7800/hardware/memory/external/elf"
	"github.com/jetsetilly/test7800/logger"
	"github.com/jetsetilly/test7800/test"
)

//go:embed "test_data/7800backgroundcolors.bin"
var testfile []byte

//go:embed "test_data/7800backgroundcolors.log"
var testfile_log []byte

type Context struct {
}

func (c *Context) Rand8Bit() uint8 {
	return 0
}

func TestELF(t *testing.T) {
	e, err := elf.NewElf(&Context{}, testfile)
	test.ExpectSuccess(t, err)
	test.ExpectInequality(t, e, nil)

	// there should be a .text section
	_, _, ok := e.ELFSection(".text")
	test.ExpectSuccess(t, ok)

	// there is no section named .foo
	_, _, ok = e.ELFSection(".foo")
	test.ExpectFailure(t, ok)

	// logging output
	b := &bytes.Buffer{}
	logger.Tail(b, -1)
	test.ExpectEquality(t, b.String(), string(testfile_log))
}
