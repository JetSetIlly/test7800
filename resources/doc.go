// Package resources contains functions to prepare paths for test7800
// resources.
//
// The JoinPath() function returns the correct path to the resource
// directory/file specified in the arguments. It handles the creation of
// directories as required but does not otherwise touch or create files.
//
// JoinPath() handles the inclusion of the correct base path. The base path
// depends on how the binary was built.
//
// For builds with the "release" build tag, the path returned by JoinPath() is
// rooted in the user's configuration directory. On modern Linux systems the
// full path would be something like:
//
//	/home/user/.config/test7800/
//
// For non-"release" builds, the correct path is rooted in the current working
// directory:
//
//	.test7800
//
// The package does this because during development it is more convenient to
// have the config directory close to hand. For release binaries however, the
// config directory should be somewhere the end-user expects.
//
// # portable.txt
//
// An exception to the above rules is when an empty file named 'portable.txt' is
// in the same directory as the Gopher2600 program binary. When the file exists
// the resources are saved in a directory named 'Gopher2600_UserData' in the
// same directory as the program binary.
package resources
