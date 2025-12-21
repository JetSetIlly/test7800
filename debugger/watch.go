package debugger

import (
	"fmt"

	"github.com/jetsetilly/test7800/hardware/memory"
)

type watch struct {
	ma    mappedAddress
	data  uint8
	prev  uint8
	write bool
}

func (m *debugger) checkWatches() (*watch, error) {
	for i, w := range m.watches {
		if w.write {
			d, err := memory.Read(w.ma.area, w.ma.idx)
			if err != nil {
				return nil, fmt.Errorf("watch: %w", err)
			}
			if d != w.data {
				w.prev = w.data
				w.data = d
				m.watches[i] = w
				return &w, nil
			}
		} else {
			if w.ma.address == m.console.Mem.LastCPUAddress && !m.console.Mem.LastCPUWrite {
				w.prev = w.data
				w.data = m.console.Mem.LastCPUData
				m.watches[i] = w
				return &w, nil
			}
		}
	}
	return nil, nil
}
