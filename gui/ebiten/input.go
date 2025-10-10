package ebiten

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/jetsetilly/test7800/gui"
)

func (eg *guiEbiten) inputGamepadAxis() error {
	const gamepad = 0
	const deadzone = 0.25

	// left and right direction of the stick
	v := ebiten.GamepadAxis(gamepad, 0)
	if eg.gamepadAnalogue[0] != 0 && v <= deadzone && v >= -deadzone {
		// stick is in the deadzone so make sure left/right input is nullified
		for _, v := range []gui.Input{{Action: gui.StickLeft}, {Action: gui.StickRight}} {
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
			case eg.g.UserInput <- gui.Input{Action: gui.StickLeft, Set: true}:
			default:
				return nil
			}
			eg.gamepadAnalogue[0] = v
		} else if v > deadzone {
			select {
			case eg.g.UserInput <- gui.Input{Action: gui.StickRight, Set: true}:
			default:
				return nil
			}
			eg.gamepadAnalogue[0] = v
		}
	}

	// up and down direction of the stick
	v = ebiten.GamepadAxis(gamepad, 1)
	if eg.gamepadAnalogue[1] != 0 && v <= deadzone && v >= -deadzone {
		for _, v := range []gui.Input{{Action: gui.StickUp}, {Action: gui.StickDown}} {
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
			case eg.g.UserInput <- gui.Input{Action: gui.StickUp, Set: true}:
			default:
				return nil
			}
			eg.gamepadAnalogue[1] = v
		} else if v > deadzone {
			select {
			case eg.g.UserInput <- gui.Input{Action: gui.StickDown, Set: true}:
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
			inp = gui.Input{Action: gui.StickLeft}
		case ebiten.GamepadButton12:
			inp = gui.Input{Action: gui.StickRight}
		case ebiten.GamepadButton11:
			inp = gui.Input{Action: gui.StickUp}
		case ebiten.GamepadButton13:
			inp = gui.Input{Action: gui.StickDown}

		// fire buttons
		case ebiten.GamepadButton0, ebiten.GamepadButton2:
			inp = gui.Input{Action: gui.StickButtonA}
		case ebiten.GamepadButton1, ebiten.GamepadButton3:
			inp = gui.Input{Action: gui.StickButtonB}

		// control
		case ebiten.GamepadButton8: // xbox button
			inp = gui.Input{Action: gui.Select}
		case ebiten.GamepadButton6: // back button
			inp = gui.Input{Action: gui.Pause}
		case ebiten.GamepadButton7: // start button
			inp = gui.Input{Action: gui.Start}
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
			inp = gui.Input{Action: gui.StickLeft, Set: true}
		case ebiten.GamepadButton12:
			inp = gui.Input{Action: gui.StickRight, Set: true}
		case ebiten.GamepadButton11:
			inp = gui.Input{Action: gui.StickUp, Set: true}
		case ebiten.GamepadButton13:
			inp = gui.Input{Action: gui.StickDown, Set: true}

		// fire buttons
		case ebiten.GamepadButton0, ebiten.GamepadButton2:
			inp = gui.Input{Action: gui.StickButtonA, Set: true}
		case ebiten.GamepadButton1, ebiten.GamepadButton3:
			inp = gui.Input{Action: gui.StickButtonB, Set: true}

		// control
		case ebiten.GamepadButton8: // xbox button
			inp = gui.Input{Action: gui.Select, Set: true}
		case ebiten.GamepadButton6: // back button
			inp = gui.Input{Action: gui.Pause, Set: true}
		case ebiten.GamepadButton7: // start button
			inp = gui.Input{Action: gui.Start, Set: true}
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
			inp = gui.Input{Action: gui.StickLeft}
		case ebiten.KeyArrowRight, ebiten.KeyNumpad6:
			inp = gui.Input{Action: gui.StickRight}
		case ebiten.KeyArrowUp, ebiten.KeyNumpad8:
			inp = gui.Input{Action: gui.StickUp}
		case ebiten.KeyArrowDown, ebiten.KeyNumpad2:
			inp = gui.Input{Action: gui.StickDown}
		case ebiten.KeySpace, ebiten.KeyZ:
			inp = gui.Input{Action: gui.StickButtonA}
		case ebiten.KeyB, ebiten.KeyX:
			inp = gui.Input{Action: gui.StickButtonB}
		case ebiten.KeyF1:
			inp = gui.Input{Action: gui.Select}
		case ebiten.KeyF2:
			inp = gui.Input{Action: gui.Start}
		case ebiten.KeyF3:
			inp = gui.Input{Action: gui.Pause}
		case ebiten.KeyF4:
			inp = gui.Input{Action: gui.P0Pro, Set: eg.proDifficulty[0]}
		case ebiten.KeyF5:
			inp = gui.Input{Action: gui.P1Pro, Set: eg.proDifficulty[1]}
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
			inp = gui.Input{Action: gui.StickLeft, Set: true}
		case ebiten.KeyArrowRight, ebiten.KeyNumpad6:
			inp = gui.Input{Action: gui.StickRight, Set: true}
		case ebiten.KeyArrowUp, ebiten.KeyNumpad8:
			inp = gui.Input{Action: gui.StickUp, Set: true}
		case ebiten.KeyArrowDown, ebiten.KeyNumpad2:
			inp = gui.Input{Action: gui.StickDown, Set: true}
		case ebiten.KeySpace, ebiten.KeyZ:
			inp = gui.Input{Action: gui.StickButtonA, Set: true}
		case ebiten.KeyB, ebiten.KeyX:
			inp = gui.Input{Action: gui.StickButtonB, Set: true}
		case ebiten.KeyF1:
			inp = gui.Input{Action: gui.Select, Set: true}
		case ebiten.KeyF2:
			inp = gui.Input{Action: gui.Start, Set: true}
		case ebiten.KeyF3:
			inp = gui.Input{Action: gui.Pause, Set: true}
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
