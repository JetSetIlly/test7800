package ui

import (
	"image"
	"io"
)

type Image struct {
	Main    *image.RGBA
	Overlay *image.RGBA
	Prev    *image.RGBA

	// the previous image includes an ID that can be used to decide if the
	// previous image has changed
	PrevID int

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
	Prefetch(n int)
}

type AudioSetup struct {
	Freq float64
	Read AudioReader
}

type UI struct {
	SetImage  chan Image
	UserInput chan Input

	// implementations of UI should default to StateRunning
	State chan State

	// AudioSetup should be nil if the emulation is to have no audio
	AudioSetup chan AudioSetup

	// optional function called by GUI during it's update loop
	UpdateGUI func() error
}

// NewUI creates a new UI instance. It does not initialise the RegisterAudio
// channel. For that, use the WithAudio() function
func NewUI() *UI {
	return &UI{
		SetImage:  make(chan Image, 1),
		UserInput: make(chan Input, 10),
		State:     make(chan State, 1),
	}
}

// WithAudio creates the RegisterAudio channel if it's not already created.
// Should not be called if the UI is to have no audio.
func (ui *UI) WithAudio() *UI {
	if ui.AudioSetup == nil {
		ui.AudioSetup = make(chan AudioSetup, 1)
	}
	return ui
}
