package gui

import (
	"image"
	"io"
)

type Image struct {
	Main    *image.RGBA
	Overlay *image.RGBA
	Prev    *image.RGBA

	// the ID of the image
	ID string

	// the x/y coordinates of the next pixel to be drawn to Main
	Cursor [2]int
}

// the state of the emulation
type State int

// the emulation state can be either paused or running
const (
	StateRunning State = iota
	StatePaused
)

type AudioReader interface {
	io.Reader
	Nudge()
}

type AudioSetup struct {
	Freq float64
	Read AudioReader
}

type GUI struct {
	SetImage  chan Image
	UserInput chan Input

	// implementations of UI should default to StateRunning
	State chan State

	// AudioSetup should be nil if the emulation is to have no audio
	AudioSetup chan AudioSetup

	// optional function called by GUI during it's update loop
	UpdateGUI func() error
}

// NewGUI creates a new GUI instance. It does not initialise the RegisterAudio
// channel. For that, use the WithAudio() function
func NewGUI() *GUI {
	return &GUI{
		SetImage:   make(chan Image, 1),
		UserInput:  make(chan Input, 10),
		State:      make(chan State, 1),
		AudioSetup: make(chan AudioSetup, 1),
	}
}
