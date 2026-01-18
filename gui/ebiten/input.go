package ebiten

import (
	"fmt"
	"io"
	"io/fs"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/jetsetilly/test7800/gui"
)

func (eg *guiEbiten) pushInput(inp gui.Input) {
	select {
	case eg.g.UserInput <- inp:
	default:
	}
}

func (eg *guiEbiten) inputDragAndDrop() error {
	df := ebiten.DroppedFiles()
	if df == nil {
		return nil
	}

	if dfs, ok := df.(fs.ReadDirFS); ok {
		fls, err := dfs.ReadDir(".")
		if err != nil {
			return err
		}
		if len(fls) > 0 {
			f, err := df.Open(fls[0].Name())
			if err != nil {
				return err
			}
			defer f.Close()
			b, err := io.ReadAll(f)
			if err != nil {
				return err
			}
			select {
			case eg.g.Blob <- gui.Blob{
				Filename: fls[0].Name(),
				Data:     b,
			}:
			default:
				return fmt.Errorf("couldn't drop file")
			}
		}
	}

	return nil
}

func (eg *guiEbiten) inputGamepadAxis() error {
	const gamepad = 0
	const deadzone = 0.25

	// left and right direction of the stick
	v := ebiten.GamepadAxis(gamepad, 0)
	if eg.gamepadAnalogue[0] != 0 && v <= deadzone && v >= -deadzone {
		// stick is in the deadzone so make sure left/right input is nullified
		for _, v := range []gui.Input{{Action: gui.StickLeft, Data: false}, {Action: gui.StickRight, Data: false}} {
			eg.pushInput(v)
		}

		// all values in the deadzone are reduced to zero
		eg.gamepadAnalogue[0] = 0

	} else if v != eg.gamepadAnalogue[0] {
		if v < -deadzone {
			eg.pushInput(gui.Input{Action: gui.StickLeft, Data: true})
			eg.gamepadAnalogue[0] = v
		} else if v > deadzone {
			eg.pushInput(gui.Input{Action: gui.StickRight, Data: true})
			eg.gamepadAnalogue[0] = v
		}
	}

	// up and down direction of the stick
	v = ebiten.GamepadAxis(gamepad, 1)
	if eg.gamepadAnalogue[1] != 0 && v <= deadzone && v >= -deadzone {
		for _, v := range []gui.Input{{Action: gui.StickUp, Data: false}, {Action: gui.StickDown, Data: false}} {
			eg.pushInput(v)
		}
		eg.gamepadAnalogue[1] = 0

	} else if v != eg.gamepadAnalogue[1] {
		if v < -deadzone {
			eg.pushInput(gui.Input{Action: gui.StickUp, Data: true})
			eg.gamepadAnalogue[1] = v
		} else if v > deadzone {
			eg.pushInput(gui.Input{Action: gui.StickDown, Data: true})
			eg.gamepadAnalogue[1] = v
		}
	}

	return nil
}

func (eg *guiEbiten) inputGamepad() error {
	var pressed []ebiten.GamepadButton
	var released []ebiten.GamepadButton
	pressed = inpututil.AppendJustPressedGamepadButtons(0, pressed)
	released = inpututil.AppendJustReleasedGamepadButtons(0, released)

	var inp gui.Input

	for _, p := range released {
		switch p {
		// d-pad
		case ebiten.GamepadButton14:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickLeft, Data: false}
		case ebiten.GamepadButton12:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickRight, Data: false}
		case ebiten.GamepadButton11:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickUp, Data: false}
		case ebiten.GamepadButton13:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickDown, Data: false}

		// fire buttons
		case ebiten.GamepadButton0, ebiten.GamepadButton2:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonA, Data: false}
		case ebiten.GamepadButton1, ebiten.GamepadButton3:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonB, Data: false}

		// control
		case ebiten.GamepadButton8: // xbox button
			inp = gui.Input{Port: gui.Panel, Action: gui.Select, Data: false}
		case ebiten.GamepadButton6: // back button
			inp = gui.Input{Port: gui.Panel, Action: gui.Pause, Data: false}
		case ebiten.GamepadButton7: // start button
			inp = gui.Input{Port: gui.Panel, Action: gui.Start, Data: false}
		}

		eg.pushInput(inp)
	}

	for _, p := range pressed {
		switch p {
		// d-pad
		case ebiten.GamepadButton14:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickLeft, Data: true}
		case ebiten.GamepadButton12:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickRight, Data: true}
		case ebiten.GamepadButton11:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickUp, Data: true}
		case ebiten.GamepadButton13:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickDown, Data: true}

		// fire buttons
		case ebiten.GamepadButton0, ebiten.GamepadButton2:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonA, Data: true}
		case ebiten.GamepadButton1, ebiten.GamepadButton3:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonB, Data: true}

		// control
		case ebiten.GamepadButton8: // xbox button
			inp = gui.Input{Port: gui.Panel, Action: gui.Select, Data: true}
		case ebiten.GamepadButton6: // back button
			inp = gui.Input{Port: gui.Panel, Action: gui.Pause, Data: true}
		case ebiten.GamepadButton7: // start button
			inp = gui.Input{Port: gui.Panel, Action: gui.Start, Data: true}
		}

		eg.pushInput(inp)
	}

	return nil
}

func (eg *guiEbiten) inputKeyboard() error {
	var pressed []ebiten.Key
	var released []ebiten.Key
	pressed = inpututil.AppendJustPressedKeys(pressed)
	released = inpututil.AppendJustReleasedKeys(released)

	var inp gui.Input

	for _, p := range released {
		switch p {
		case ebiten.KeyEscape:
			return ebiten.Termination
		case ebiten.KeyArrowLeft, ebiten.KeyNumpad4:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickLeft, Data: false}
		case ebiten.KeyArrowRight, ebiten.KeyNumpad6:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickRight, Data: false}
		case ebiten.KeyArrowUp, ebiten.KeyNumpad8:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickUp, Data: false}
		case ebiten.KeyArrowDown, ebiten.KeyNumpad2:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickDown, Data: false}
		case ebiten.KeySpace, ebiten.KeyZ:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonA, Data: false}
		case ebiten.KeyB, ebiten.KeyX:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonB, Data: false}
		case ebiten.KeyF1:
			inp = gui.Input{Port: gui.Panel, Action: gui.Select, Data: false}
		case ebiten.KeyF2:
			inp = gui.Input{Port: gui.Panel, Action: gui.Start, Data: false}
		case ebiten.KeyF3:
			inp = gui.Input{Port: gui.Panel, Action: gui.Pause, Data: false}
		case ebiten.KeyF4:
			inp = gui.Input{Port: gui.Panel, Action: gui.P0Pro, Data: eg.proDifficulty[0]}
		case ebiten.KeyF5:
			inp = gui.Input{Port: gui.Panel, Action: gui.P1Pro, Data: eg.proDifficulty[1]}

		case ebiten.KeyF7:
			eg.showInfo = !eg.showInfo
		case ebiten.KeyF11:
			eg.geom.fullScreen = !eg.geom.fullScreen
			ebiten.SetFullscreen(eg.geom.fullScreen)
		}

		eg.pushInput(inp)
	}

	for _, r := range pressed {
		switch r {
		case ebiten.KeyArrowLeft, ebiten.KeyNumpad4:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickLeft, Data: true}
		case ebiten.KeyArrowRight, ebiten.KeyNumpad6:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickRight, Data: true}
		case ebiten.KeyArrowUp, ebiten.KeyNumpad8:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickUp, Data: true}
		case ebiten.KeyArrowDown, ebiten.KeyNumpad2:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickDown, Data: true}
		case ebiten.KeySpace, ebiten.KeyZ:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonA, Data: true}
		case ebiten.KeyB, ebiten.KeyX:
			inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonB, Data: true}
		case ebiten.KeyF1:
			inp = gui.Input{Port: gui.Panel, Action: gui.Select, Data: true}
		case ebiten.KeyF2:
			inp = gui.Input{Port: gui.Panel, Action: gui.Start, Data: true}
		case ebiten.KeyF3:
			inp = gui.Input{Port: gui.Panel, Action: gui.Pause, Data: true}
		case ebiten.KeyF4:
			eg.proDifficulty[0] = !eg.proDifficulty[0]
		case ebiten.KeyF5:
			eg.proDifficulty[1] = !eg.proDifficulty[1]
		}

		eg.pushInput(inp)
	}

	return nil
}

func isCursorInWindow() bool {
	if !ebiten.IsFocused() {
		return false
	}
	x, y := ebiten.CursorPosition()
	w, h := ebiten.WindowSize()
	return x >= 0 && y >= 0 && x < w && y < h
}

func (eg *guiEbiten) inputMouse() error {
	if eg.mouseCaptured {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
			eg.mouseCaptured = false
			eg.pushInput(gui.Input{Port: gui.Undefined, Action: gui.AnalogueSelect, Data: false})
		}
	} else if isCursorInWindow() {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {
			ebiten.SetCursorMode(ebiten.CursorModeCaptured)
			eg.mouseCaptured = true
			eg.pushInput(gui.Input{Port: gui.Undefined, Action: gui.AnalogueSelect, Data: true})

			// update mouse position reference immediately so that we don't push erroneous movement
			// to the emulation
			eg.mouseX, eg.mouseY = ebiten.CursorPosition()
		}
	}

	if !eg.mouseCaptured {
		return nil
	}

	// function to change the mouse movement acceleration
	negativeAcceleration := func(delta float64, exp float64) float64 {
		return math.Copysign(math.Pow(math.Abs(delta), exp), delta)
	}

	// movement deltas and recording current mouse position for next frame
	x, y := ebiten.CursorPosition()
	dx := x - eg.mouseX
	dy := y - eg.mouseY
	eg.mouseX = x
	eg.mouseY = y

	// trakball movement
	if dx != 0 || dy != 0 {
		eg.pushInput(gui.Input{Port: gui.Undefined, Action: gui.TrakballMove, Data: gui.TrakballMoveData{
			DeltaX: dx,
			DeltaY: dy,
		}})
	}

	const paddleExp = 0.6
	dx = int(negativeAcceleration(float64(dx), paddleExp))
	dy = int(negativeAcceleration(float64(dy), paddleExp))

	// mix y-axis with x-axis. in this scenario the absolute value of the y-axis
	// is given the same sign as the x-axis
	delta := dx
	if dy < 0 {
		dy *= -1
	}
	if dx < 0 {
		delta -= dy
	} else if x > 0 {
		delta += dy
	}

	// paddle movement
	if delta != 0 {
		eg.pushInput(gui.Input{Port: gui.Undefined, Action: gui.PaddleMove, Data: gui.PaddleMoveData{
			Paddle: 0,
			Delta:  delta,
		}})
	}

	// fire buttons for paddle and trakball
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
		eg.pushInput(gui.Input{Port: gui.Undefined, Action: gui.PaddleFire, Data: gui.PaddleFireData{
			Paddle: 0,
			Fire:   true,
		}})
		eg.pushInput(gui.Input{Port: gui.Undefined, Action: gui.TrakballFire, Data: true})
	} else if inpututil.IsMouseButtonJustReleased(ebiten.MouseButton0) {
		eg.pushInput(gui.Input{Port: gui.Undefined, Action: gui.PaddleFire, Data: gui.PaddleFireData{
			Paddle: 0,
			Fire:   false,
		}})
		eg.pushInput(gui.Input{Port: gui.Undefined, Action: gui.TrakballFire, Data: false})
	}

	return nil
}
