package debugger

import (
	"github.com/jetsetilly/test7800/coprocessor"
)

func (m *debugger) CartYield(yld coprocessor.CoProcYield) coprocessor.YieldHookResponse {
	if yld.Type.Normal() {
		return coprocessor.YieldHookContinue
	}
	return coprocessor.YieldHookEnd
}
