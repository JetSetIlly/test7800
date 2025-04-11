package debugger

import "github.com/jetsetilly/test7800/coprocessor"

type coprocDisasm struct {
	enabled bool
	last    []coprocessor.CartCoProcDisasmEntry
	summary coprocessor.CartCoProcDisasmSummary
}

// Start is called at the beginning of coprocessor program execution.
func (dsm *coprocDisasm) Start() {
	dsm.last = dsm.last[:0]
}

// Step called after every instruction in the coprocessor program.
func (dsm *coprocDisasm) Step(e coprocessor.CartCoProcDisasmEntry) {
	dsm.last = append(dsm.last, e)
}

// End is called when coprocessor program has finished.
func (dsm *coprocDisasm) End(summary coprocessor.CartCoProcDisasmSummary) {
	dsm.summary = summary
}
