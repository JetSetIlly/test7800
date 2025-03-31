package io

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
)
