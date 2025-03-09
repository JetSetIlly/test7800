package main

import (
	"fmt"

	"github.com/jetsetilly/test7800/debugger"
	"github.com/jetsetilly/test7800/gui"
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

	go func() {
		resultGui <- gui.Launch(endGui)
		endDebugger <- true
	}()

	go func() {
		resultDebugger <- debugger.Launch(endDebugger)
		endGui <- true
	}()

	if err := <-resultGui; err != nil {
		fmt.Println(err)
	}
	if err := <-resultDebugger; err != nil {
		fmt.Println(err)
	}
}
