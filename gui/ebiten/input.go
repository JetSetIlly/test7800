package ebiten

import (
	"fmt"
	"io"
	"io/fs"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/jetsetilly/test7800/gui"
)

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
			select {
			case eg.g.UserInput <- v:
			default:
				return nil
			}
		}

		// all values in the deadzone are reduced to zero
		eg.gamepadAnalogue[0] = 0

	} else if v != eg.gamepadAnalogue[0] {
		if v < -deadzone {
			select {
			case eg.g.UserInput <- gui.Input{Action: gui.StickLeft, Data: true}:
			default:
				return nil
			}
			eg.gamepadAnalogue[0] = v
		} else if v > deadzone {
			select {
			case eg.g.UserInput <- gui.Input{Action: gui.StickRight, Data: true}:
			default:
				return nil
			}
			eg.gamepadAnalogue[0] = v
		}
	}

	// up and down direction of the stick
	v = ebiten.GamepadAxis(gamepad, 1)
	if eg.gamepadAnalogue[1] != 0 && v <= deadzone && v >= -deadzone {
		for _, v := range []gui.Input{{Action: gui.StickUp, Data: false}, {Action: gui.StickDown, Data: false}} {
			select {
			case eg.g.UserInput <- v:
			default:
				return nil
			}
		}
		eg.gamepadAnalogue[1] = 0

	} else if v != eg.gamepadAnalogue[1] {
		if v < -deadzone {
			select {
			case eg.g.UserInput <- gui.Input{Action: gui.StickUp, Data: true}:
			default:
				return nil
			}
			eg.gamepadAnalogue[1] = v
		} else if v > deadzone {
			select {
			case eg.g.UserInput <- gui.Input{Action: gui.StickDown, Data: true}:
			default:
				return nil
			}
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

		select {
		case eg.g.UserInput <- inp:
		default:
			return nil
		}
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

		select {
		case eg.g.UserInput <- inp:
		default:
			return nil
		}
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
		}

		select {
		case eg.g.UserInput <- inp:
		default:
			return nil
		}
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

		select {
		case eg.g.UserInput <- inp:
		default:
			return nil
		}
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
		}
	} else if isCursorInWindow() {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {
			ebiten.SetCursorMode(ebiten.CursorModeCaptured)
			eg.mouseCaptured = true
		}
	}

	if !eg.mouseCaptured {
		return nil
	}

	x, y := ebiten.CursorPosition()
	deltaX := x - eg.mouseX
	deltaY := y - eg.mouseY
	eg.mouseX = x
	eg.mouseY = y

	if deltaX != 0 || deltaY != 0 {
		fmt.Println(deltaX, deltaY)
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
		inp := gui.Input{Port: gui.Player0, Action: gui.PaddleFire, Data: true}
		select {
		case eg.g.UserInput <- inp:
		default:
			return nil
		}
	} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
		inp := gui.Input{Port: gui.Player0, Action: gui.PaddleFire, Data: false}
		select {
		case eg.g.UserInput <- inp:
		default:
			return nil
		}
	}

	return nil
}
