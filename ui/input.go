package ui

type Action int

type Input struct {
	Action Action
	Set    bool
}

const (
	Nothing Action = iota
	StickLeft
	StickUp
	StickRight
	StickDown
	StickButtonA
	StickButtonB
	Select
	Start
	Pause
	P0Pro
	P1Pro
)
