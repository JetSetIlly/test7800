package gui

type Action int

type Port int

type Source string

type Input struct {
	Port   Port
	Action Action
	Data   any
	Source Source
}

const (
	Player0 Port = iota
	Player1
	Panel
)

const Undefined Port = -1

type PaddleFireData struct {
	Paddle int // 0 or 1 to indicate which paddle in the pair
	Fire   bool
}

type PaddleMoveData struct {
	Paddle int // 0 or 1 to indicate which paddle in the pair
	Delta  int // distance moved by paddle device
}

type TrakballMoveData struct {
	DeltaX int
	DeltaY int
}

const (
	Nothing Action = iota

	Select // bool
	Start  // bool
	Pause  // bool
	P0Pro  // bool
	P1Pro  // bool

	StickLeft    // bool
	StickUp      // bool
	StickRight   // bool
	StickDown    // bool
	StickButtonA // bool
	StickButtonB // bool

	AnalogueSelect // bool

	PaddleFire // PaddleFireData
	PaddleMove // PaddleMoveData

	TrakballFire // bool
	TrakballMove // TrakballMove
)
