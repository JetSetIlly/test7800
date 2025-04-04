package ui

import (
	"image"
	"io"
)

type UI struct {
	SetImage      chan *image.RGBA
	RegisterAudio chan io.Reader
	UserInput     chan Input
}

func NewUI() *UI {
	return &UI{
		SetImage:      make(chan *image.RGBA, 1),
		RegisterAudio: make(chan io.Reader, 1),
		UserInput:     make(chan Input, 1),
	}
}
