package ebiten

import (
	"fmt"
	"io"
	"io/fs"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/logger"
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

func (eg *guiEbiten) detectKeyboards() error {
	// just one keyboard supported for now
	if len(eg.keyboards) == 0 {
		eg.keyboards = append(eg.keyboards, gui.InputSource{
			Type: gui.InputKeyboard,
			Name: "keyboard",
			ID:   0,
		})
	}

	return nil
}

func (eg *guiEbiten) detectGamepads() error {
	var buf []ebiten.GamepadID
	buf = inpututil.AppendJustConnectedGamepadIDs(buf)

	for _, id := range buf {
		if ebiten.IsStandardGamepadLayoutAvailable(id) {
			source := gui.InputSource{
				Type: gui.InputGamepad,
				Name: ebiten.GamepadName(id),
				ID:   int(id),
			}
			eg.gamepads = append(eg.gamepads, source)
			eg.gamepadAnalogue = append(eg.gamepadAnalogue, [2]float64{})
			logger.Logf(logger.Allow, "gamepad", "attached %s", source.Name)
		}
	}

	// delete entry from gamepads and corresponding gamepadAnalogue entry if a gamepad has been
	// disconnected. being careful to keep the two slices in sync
	i := 0
	for j, source := range eg.gamepads {
		if inpututil.IsGamepadJustDisconnected(ebiten.GamepadID(source.ID)) {
			logger.Logf(logger.Allow, "gamepad", "detached %s", source.Name)
		} else {
			eg.gamepads[i] = eg.gamepads[j]
			eg.gamepadAnalogue[i] = eg.gamepadAnalogue[j]
			i++
		}
	}
	eg.gamepads = eg.gamepads[:i]
	eg.gamepadAnalogue = eg.gamepadAnalogue[:i]

	return nil
}

func (eg *guiEbiten) inputGamepadAxis() error {
	const deadzone = 0.25

	for i, source := range eg.gamepads {
		// left and right direction of the stick
		v := ebiten.GamepadAxisValue(ebiten.GamepadID(source.ID), 0)
		if eg.gamepadAnalogue[i][0] != 0 && v <= deadzone && v >= -deadzone {
			// stick is in the deadzone so make sure left/right input is nullified
			nullify := []gui.Input{
				{Action: gui.StickLeft, Data: false, Source: source},
				{Action: gui.StickRight, Data: false, Source: source},
			}
			for _, v := range nullify {
				eg.pushInput(v)
			}
			eg.gamepadAnalogue[i][0] = 0

		} else if v != eg.gamepadAnalogue[i][0] {
			if v < -deadzone {
				eg.pushInput(gui.Input{Action: gui.StickLeft, Data: true, Source: source})
				eg.gamepadAnalogue[i][0] = v
			} else if v > deadzone {
				eg.pushInput(gui.Input{Action: gui.StickRight, Data: true, Source: source})
				eg.gamepadAnalogue[i][0] = v
			}
		}

		// up and down direction of the stick
		v = ebiten.GamepadAxisValue(ebiten.GamepadID(source.ID), 1)
		if eg.gamepadAnalogue[i][1] != 0 && v <= deadzone && v >= -deadzone {
			// stick is in the deadzone so make sure left/right input is nullified
			nullify := []gui.Input{
				{Action: gui.StickUp, Data: false, Source: source},
				{Action: gui.StickDown, Data: false, Source: source},
			}
			for _, v := range nullify {
				eg.pushInput(v)
			}
			eg.gamepadAnalogue[i][1] = 0

		} else if v != eg.gamepadAnalogue[i][1] {
			if v < -deadzone {
				eg.pushInput(gui.Input{Action: gui.StickUp, Data: true, Source: source})
				eg.gamepadAnalogue[i][1] = v
			} else if v > deadzone {
				eg.pushInput(gui.Input{Action: gui.StickDown, Data: true, Source: source})
				eg.gamepadAnalogue[i][1] = v
			}
		}
	}

	return nil
}

func (eg *guiEbiten) inputGamepad() error {
	for _, source := range eg.gamepads {
		var pressed []ebiten.StandardGamepadButton
		var released []ebiten.StandardGamepadButton
		pressed = inpututil.AppendJustPressedStandardGamepadButtons(ebiten.GamepadID(source.ID), pressed)
		released = inpututil.AppendJustReleasedStandardGamepadButtons(ebiten.GamepadID(source.ID), released)

		var inp gui.Input

		for _, p := range released {
			switch p {
			// d-pad
			case ebiten.StandardGamepadButtonLeftLeft:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickLeft, Data: false, Source: source}
			case ebiten.StandardGamepadButtonLeftRight:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickRight, Data: false, Source: source}
			case ebiten.StandardGamepadButtonLeftTop:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickUp, Data: false, Source: source}
			case ebiten.StandardGamepadButtonLeftBottom:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickDown, Data: false, Source: source}

			// fire buttons
			case ebiten.StandardGamepadButtonRightBottom, ebiten.StandardGamepadButtonRightLeft:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonA, Data: false, Source: source}
			case ebiten.StandardGamepadButtonRightRight, ebiten.StandardGamepadButtonRightTop:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonB, Data: false, Source: source}

			// control
			case ebiten.StandardGamepadButtonCenterCenter: // xbox button
				inp = gui.Input{Port: gui.Panel, Action: gui.Select, Data: false, Source: source}
			case ebiten.StandardGamepadButtonCenterLeft: // back button
				inp = gui.Input{Port: gui.Panel, Action: gui.Pause, Data: false, Source: source}
			case ebiten.StandardGamepadButtonCenterRight: // start button
				inp = gui.Input{Port: gui.Panel, Action: gui.Start, Data: false, Source: source}
			}
			eg.pushInput(inp)
		}

		for _, p := range pressed {
			switch p {
			// d-pad
			case ebiten.StandardGamepadButtonLeftLeft:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickLeft, Data: true, Source: source}
			case ebiten.StandardGamepadButtonLeftRight:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickRight, Data: true, Source: source}
			case ebiten.StandardGamepadButtonLeftTop:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickUp, Data: true, Source: source}
			case ebiten.StandardGamepadButtonLeftBottom:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickDown, Data: true, Source: source}

				// fire buttons
			case ebiten.StandardGamepadButtonRightBottom, ebiten.StandardGamepadButtonRightLeft:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonA, Data: true, Source: source}
			case ebiten.StandardGamepadButtonRightRight, ebiten.StandardGamepadButtonRightTop:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonB, Data: true, Source: source}

			// control
			case ebiten.StandardGamepadButtonCenterCenter: // xbox button
				inp = gui.Input{Port: gui.Panel, Action: gui.Select, Data: true, Source: source}
			case ebiten.StandardGamepadButtonCenterLeft: // xbox button
				inp = gui.Input{Port: gui.Panel, Action: gui.Pause, Data: true, Source: source}
			case ebiten.StandardGamepadButtonCenterRight: // xbox button
				inp = gui.Input{Port: gui.Panel, Action: gui.Start, Data: true, Source: source}

			}
			eg.pushInput(inp)
		}
	}

	return nil
}

func (eg *guiEbiten) inputKeyboard() error {
	for _, source := range eg.keyboards {
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
				inp = gui.Input{Port: gui.Player0, Action: gui.StickLeft, Data: false, Source: source}
			case ebiten.KeyArrowRight, ebiten.KeyNumpad6:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickRight, Data: false, Source: source}
			case ebiten.KeyArrowUp, ebiten.KeyNumpad8:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickUp, Data: false, Source: source}
			case ebiten.KeyArrowDown, ebiten.KeyNumpad2:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickDown, Data: false, Source: source}
			case ebiten.KeySpace, ebiten.KeyZ:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonA, Data: false, Source: source}
			case ebiten.KeyB, ebiten.KeyX:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonB, Data: false, Source: source}
			case ebiten.KeyF1:
				inp = gui.Input{Port: gui.Panel, Action: gui.Select, Data: false, Source: source}
			case ebiten.KeyF2:
				inp = gui.Input{Port: gui.Panel, Action: gui.Start, Data: false, Source: source}
			case ebiten.KeyF3:
				inp = gui.Input{Port: gui.Panel, Action: gui.Pause, Data: false, Source: source}
			case ebiten.KeyF4:
				inp = gui.Input{Port: gui.Panel, Action: gui.P0Pro, Data: eg.proDifficulty[0], Source: source}
			case ebiten.KeyF5:
				inp = gui.Input{Port: gui.Panel, Action: gui.P1Pro, Data: eg.proDifficulty[1], Source: source}

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
				inp = gui.Input{Port: gui.Player0, Action: gui.StickLeft, Data: true, Source: source}
			case ebiten.KeyArrowRight, ebiten.KeyNumpad6:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickRight, Data: true, Source: source}
			case ebiten.KeyArrowUp, ebiten.KeyNumpad8:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickUp, Data: true, Source: source}
			case ebiten.KeyArrowDown, ebiten.KeyNumpad2:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickDown, Data: true, Source: source}
			case ebiten.KeySpace, ebiten.KeyZ:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonA, Data: true, Source: source}
			case ebiten.KeyB, ebiten.KeyX:
				inp = gui.Input{Port: gui.Player0, Action: gui.StickButtonB, Data: true, Source: source}
			case ebiten.KeyF1:
				inp = gui.Input{Port: gui.Panel, Action: gui.Select, Data: true, Source: source}
			case ebiten.KeyF2:
				inp = gui.Input{Port: gui.Panel, Action: gui.Start, Data: true, Source: source}
			case ebiten.KeyF3:
				inp = gui.Input{Port: gui.Panel, Action: gui.Pause, Data: true, Source: source}
			case ebiten.KeyF4:
				eg.proDifficulty[0] = !eg.proDifficulty[0]
			case ebiten.KeyF5:
				eg.proDifficulty[1] = !eg.proDifficulty[1]
			}

			eg.pushInput(inp)
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
