package gui

type Action int

type Port int

type InputType int

const (
	InputAny InputType = iota
	InputKeyboard
	InputGamepad
	InputNone
)

func (t InputType) GoString() string {
	switch t {
	case InputNone:
		return "none"
	case InputKeyboard:
		return "keyboard"
	case InputGamepad:
		return "gamepad"
	case InputAny:
		return "any"
	}
	return "unknown"
}

type InputSource struct {
	Type     InputType
	Position int
}

type Input struct {
	Source InputSource
	Port   Port
	Action Action
	Data   any
}

const (
	Players Port = iota
	Panel
)

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
