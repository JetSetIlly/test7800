package gui

type Action int

type Port int

type Input struct {
	Port   Port
	Action Action
	Data   any
}

const (
	Unplugged Port = iota
	Player0
	Player1
	Panel
)

const (
	Nothing Action = iota

	Select
	Start
	Pause
	P0Pro
	P1Pro

	StickLeft
	StickUp
	StickRight
	StickDown
	StickButtonA
	StickButtonB

	PaddleFire
	PaddleSet
)
