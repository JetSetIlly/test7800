package debugger

import (
	"fmt"

	"github.com/jetsetilly/test7800/hardware/memory"
)

type watch struct {
	ma   mappedAddress
	data uint8
	prev uint8
}

func (m *debugger) checkWatches() (*watch, error) {
	for i, w := range m.watches {
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
	}
	return nil, nil
}
