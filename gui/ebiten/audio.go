package ebiten

import (
	"sync"

	"github.com/ebitengine/oto/v3"
	"github.com/jetsetilly/test7800/gui"
)

type audioPlayer struct {
	p *oto.Player
	r gui.AudioReader

	// the state field is accessed by the Read() function via the audio
	// engine, and by the GUI which is in another goroutine. access to the state
	// field therefore, is proctected by a mutex
	crit  sync.Mutex
	state gui.State
}

func (a *audioPlayer) setState(state gui.State) {
	a.crit.Lock()
	defer a.crit.Unlock()
	a.state = state
	if a.p != nil {
		if state == gui.StatePaused {
			a.p.Pause()
		} else {
			a.p.Play()
		}
	}
}

func (a *audioPlayer) Read(buf []uint8) (int, error) {
	a.crit.Lock()
	defer a.crit.Unlock()
	if a.state != gui.StateRunning {
		return 0, nil
	}

	const prefetch = 2048

	sz := a.p.BufferedSize()
	if sz < prefetch {
		a.r.Nudge()
	}

	n, err := a.r.Read(buf)
	if err != nil {
		return 0, err
	}
	return n, nil
}
