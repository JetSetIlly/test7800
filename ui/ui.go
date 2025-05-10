package ui

import (
	"image"
	"io"
)

type Image struct {
	Main    *image.RGBA
	Overlay *image.RGBA
}

type UI struct {
	SetImage  chan Image
	UserInput chan Input

	// RegisterAudio should be nil if the emulation is to have no audio
	RegisterAudio chan io.Reader

	// optional function called by GUI during it's update loop
	UpdateGUI func() error
}

// NewUI creates a new UI instance. It does not initialise the RegisterAudio
// channel. For that, use the WithAudio() function
func NewUI() *UI {
	return &UI{
		SetImage:  make(chan Image, 1),
		UserInput: make(chan Input, 10),
	}
}

// WithAudio creates the RegisterAudio channel if it's not already created.
// Should not be called if the UI is to have no audio.
func (ui *UI) WithAudio() *UI {
	if ui.RegisterAudio == nil {
		ui.RegisterAudio = make(chan io.Reader, 1)
	}
	return ui
}
