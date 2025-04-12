package debugger

import (
	"github.com/jetsetilly/test7800/coprocessor"
	"github.com/jetsetilly/test7800/coprocessor/faults"
)

type coprocDev struct {
	faults faults.Faults
}

func newCoprocDev() *coprocDev {
	return &coprocDev{
		faults: faults.NewFaults(),
	}
}

// a memory fault has occured
func (dev *coprocDev) MemoryFault(event string, explanation faults.Category, instructionAddr uint32, accessAddr uint32) {
	dev.faults.NewEntry(event, explanation, instructionAddr, accessAddr)
}

// returns the highest address used by the program. the coprocessor uses
// this value to detect stack collisions
func (dev *coprocDev) HighAddress() uint32 {
	return 0
}

// checks if address has a breakpoint assigned to it
func (dev *coprocDev) CheckBreakpoint(addr uint32) bool {
	return false
}

// returns a map that can be used to count cycles for each PC address
func (dev *coprocDev) Profiling() *coprocessor.CartCoProcProfiler {
	return nil
}

// notifies developer that the start of a new profiling session is about to begin
func (dev *coprocDev) StartProfiling() {
}

// instructs developer implementation to accumulate profiling data. there
// can be many calls to profiling profiling for every call to start
// profiling
func (dev *coprocDev) ProcessProfiling() {
}

// called whenever the ARM yields to the VCS. it communicates the address of
// the most recent instruction and the reason for the yield
func (dev *coprocDev) OnYield(addr uint32, reason coprocessor.CoProcYield) {
}
