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

// the emulation state can be either paused or running. should default to state running
const (
	StateRunning State = iota
	StatePaused
)

type AudioReader interface {
	io.Reader
	Nudge()
}

type AudioSetup struct {
	Freq int
	Read AudioReader
}

type Command []string

type Blob struct {
	Filename string
	Data     []uint8
}

type Channels struct {
	SetImage  chan Image
	UserInput chan Input
	Commands  chan Command
	Blob      chan Blob

	// implementations of UI should default to StateRunning
	State chan State

	// AudioSetup should be nil if the emulation is to have no audio
	AudioSetup chan AudioSetup

	// gui receives a string (last selected file) over the FileRequest channel
	// and returns result over the RequestedFile channel
	FileRequest   chan string
	RequestedFile chan string

	// display an error message
	ErrorDialog chan string
}

type ChannelsGUI struct {
	SetImage      <-chan Image
	UserInput     chan<- Input
	Commands      chan<- Command
	Blob          chan<- Blob
	State         <-chan State
	AudioSetup    <-chan AudioSetup
	FileRequest   <-chan string
	RequestedFile chan<- string
	ErrorDialog   <-chan string
}

type ChannelsDebugger struct {
	SetImage      chan<- Image
	UserInput     <-chan Input
	Commands      <-chan Command
	Blob          <-chan Blob
	State         chan<- State
	AudioSetup    chan<- AudioSetup
	FileRequest   chan<- string
	RequestedFile <-chan string
	ErrorDialog   chan<- string
}

func (c *Channels) GUI() *ChannelsGUI {
	return &ChannelsGUI{
		SetImage:      c.SetImage,
		UserInput:     c.UserInput,
		Commands:      c.Commands,
		Blob:          c.Blob,
		State:         c.State,
		AudioSetup:    c.AudioSetup,
		FileRequest:   c.FileRequest,
		RequestedFile: c.RequestedFile,
		ErrorDialog:   c.ErrorDialog,
	}
}

func (c *Channels) Debugger() *ChannelsDebugger {
	return &ChannelsDebugger{
		SetImage:      c.SetImage,
		UserInput:     c.UserInput,
		Commands:      c.Commands,
		Blob:          c.Blob,
		State:         c.State,
		AudioSetup:    c.AudioSetup,
		FileRequest:   c.FileRequest,
		RequestedFile: c.RequestedFile,
		ErrorDialog:   c.ErrorDialog,
	}
}

func NewChannels() *Channels {
	return &Channels{
		SetImage:      make(chan Image, 1),
		UserInput:     make(chan Input, 10),
		Commands:      make(chan Command, 10),
		Blob:          make(chan Blob, 1),
		State:         make(chan State, 1),
		AudioSetup:    make(chan AudioSetup, 1),
		FileRequest:   make(chan string, 1),
		RequestedFile: make(chan string, 1),
		ErrorDialog:   make(chan string, 1),
	}
}
