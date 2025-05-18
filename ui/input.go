package ui

type Action int

type Input struct {
	Action  Action
	Release bool
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
	Reset
	P0Pro
	P1Pro
)
