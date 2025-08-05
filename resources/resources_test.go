package resources_test

import (
	"testing"

	"github.com/jetsetilly/test7800/resources"
	"github.com/jetsetilly/test7800/test"
)

func TestJoinPath(t *testing.T) {
	pth, err := resources.JoinPath("foo/bar", "baz")
	test.ExpectEquality(t, err, nil)
	test.ExpectEquality(t, pth, ".test7800/foo/bar/baz")

	pth, err = resources.JoinPath("foo", "bar", "baz")
	test.ExpectEquality(t, err, nil)
	test.ExpectEquality(t, pth, ".test7800/foo/bar/baz")

	pth, err = resources.JoinPath("foo/bar", "")
	test.ExpectEquality(t, err, nil)
	test.ExpectEquality(t, pth, ".test7800/foo/bar")

	pth, err = resources.JoinPath("", "baz")
	test.ExpectEquality(t, err, nil)
	test.ExpectEquality(t, pth, ".test7800/baz")

	pth, err = resources.JoinPath("", "")
	test.ExpectEquality(t, err, nil)
	test.ExpectEquality(t, pth, ".test7800")
}
