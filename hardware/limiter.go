package hardware

import (
	"time"

	"github.com/jetsetilly/test7800/hardware/spec"
)

type limiter struct {
	tick  *time.Ticker
	nudge chan bool

	// the payload function for the Wait() method
	wait func()
}

func newLimiter(spec spec.Spec) *limiter {
	// the nominal speed of the console
	hz := spec.HorizScan / float64(spec.AbsoluteBottom)

	l := &limiter{
		tick:  time.NewTicker(time.Second / time.Duration(hz)),
		nudge: make(chan bool, 1),
	}

	// the wait() function changes after a few nudges. this helps ensure that the audio and video
	// synchronise after startup. the technique employed here simplifies the code path of the wait()
	// function once the emulation is up-and-running
	var ct int
	l.wait = func() {
		select {
		case <-time.After(20 * time.Millisecond):
		case <-l.nudge:
			ct++
			if ct > 2 {
				l.wait = func() {
					select {
					case <-l.tick.C:
					case <-l.nudge:
					}
				}
			}
		}
	}

	return l
}

func (l *limiter) Wait() {
	l.wait()
}

func (l *limiter) Nudge() {
	select {
	case l.nudge <- true:
	default:
	}
}
