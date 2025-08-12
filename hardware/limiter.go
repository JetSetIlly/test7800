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
	l := &limiter{
		nudge: make(chan bool, 1),
	}

	// the ideal speed of the console
	hz := spec.HorizScan / float64(spec.AbsoluteBottom)
	d := time.Second / time.Duration(hz)

	// the wait() function deliberatey starts slow and then changes state after a few nudges to
	// normal operation
	//
	// this helps ensure that the audio and video synchronise after startup
	var ct int
	l.wait = func() {
		select {
		case <-time.After(time.Duration(float64(d) * 1.025)):
		case <-l.nudge:
			ct++
			if ct > 2 {
				l.tick = time.NewTicker(d)
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
