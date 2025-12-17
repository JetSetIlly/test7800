package main

import (
	"fmt"
	"os"

	"github.com/jetsetilly/test7800/debugger"
	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/gui/ebiten"
)

func main() {
	// buffered channels. this means we don't have to worry about the gui closing
	// before the debugger and vice versa
	endGui := make(chan bool, 1)
	endDebugger := make(chan bool, 1)

	g := gui.NewGUI()

	go func() {
		err := debugger.Launch(endDebugger, g, os.Args[1:])
		if err != nil {
			fmt.Printf("*** %s\n", err)
		}
		endGui <- true
	}()

	err := ebiten.Launch(endGui, g)
	if err != nil {
		fmt.Printf("*** %s\n", err)
	}
	endDebugger <- true
}
