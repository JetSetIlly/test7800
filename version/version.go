package version

import (
	"fmt"
	"runtime/debug"
)

// The name to use when referring to the application
const ApplicationName = "Test7800"

// if number is empty then the project was probably not built using the makefile
var number string

// Revision contains the vcs revision. If the source has been modified but
// has not been committed then the Revision string will be suffixed with
// "+dirty"
var revision string

// Version contains a the current version number of the project
//
// If the version string is "unreleased" then it means that the project has
// been manually built (ie. not with the makefile)
//
// If the version string is "local" then it means that there is no no version
// number and no vcs information. This can happen when compiling/running with
// "go run ."
var version string

// Version returns the version string, the revision string and whether this is a
// numbered "release" version. if release is true then the revision information
// should be used sparingly
func Version() (string, string, bool) {
	return version, revision, version == number
}

// Title returns a string that can be used in a window title. It concatenates the application name
// with the version number depending on whether the program is a "release" version
func Title() string {
	var title string
	ver, rev, rel := Version()
	if rel {
		title = fmt.Sprintf("%s (%s)", ApplicationName, ver)
	} else {
		title = fmt.Sprintf("%s (%s)", ApplicationName, rev)
	}
	return title
}

func init() {
	var vcs bool
	var vcsRevision string
	var vcsModified bool

	info, ok := debug.ReadBuildInfo()
	if ok {
		for _, v := range info.Settings {
			switch v.Key {
			case "vcs":
				vcs = true
			case "vcs.revision":
				vcsRevision = v.Value
			case "vcs.modified":
				switch v.Value {
				case "true":
					vcsModified = true
				default:
					vcsModified = false
				}
			}
		}
	}

	if vcsRevision == "" {
		revision = "no revision information"
	} else {
		revision = vcsRevision
		if vcsModified {
			revision = fmt.Sprintf("%s+dirty", revision)
		}
	}

	if number == "" {
		if vcs {
			version = "unreleased"
		} else {
			version = "local"
		}
	} else {
		version = number
	}
}
