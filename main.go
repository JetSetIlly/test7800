package main

import (
	"fmt"
	"os"

	"github.com/jetsetilly/test7800/debugger"
	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/ui"
)

func main() {
	var endGui chan bool
	var endDebugger chan bool
	var resultGui chan error
	var resultDebugger chan error

	// buffered channels. this means we don't have to worry about the gui closing
	// before the debugger and vice versa
	endGui = make(chan bool, 1)
	endDebugger = make(chan bool, 1)

	// similarly, the result channels are buffered because we don't know the
	// order in which the gui and debugger will end
	resultGui = make(chan error, 1)
	resultDebugger = make(chan error, 1)

	u := ui.NewUI().WithAudio()

	go func() {
		resultGui <- gui.Launch(endGui, u)
		endDebugger <- true
	}()

	go func() {
		resultDebugger <- debugger.Launch(endDebugger, u, os.Args[1:])
		endGui <- true
	}()

	if err := <-resultGui; err != nil {
		fmt.Printf("*** %s\n", err)
	}
	if err := <-resultDebugger; err != nil {
		fmt.Printf("*** %s\n", err)
	}
}
