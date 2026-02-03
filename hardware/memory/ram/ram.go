package ram

import (
	"fmt"
	"strings"
)

type RAM struct {
	ctx   Context
	label string
	data  []uint8
}

type Context interface {
	Rand8Bit() uint8
}

func Create(ctx Context, label string, size int) *RAM {
	return &RAM{
		ctx:   ctx,
		label: label,
		data:  make([]uint8, size),
	}
}

func (r *RAM) Reset(random bool) {
	if random {
		for i := range len(r.data) {
			r.data[i] = r.ctx.Rand8Bit()
		}
	} else {
		clear(r.data)
	}
}

func (r *RAM) String() string {
	var s strings.Builder
	for i := 0; i <= (len(r.data)-1)/16; i++ {
		j := i * 16
		fmt.Fprintf(&s, "%04x : % 02x\n", j, r.data[j:j+16])
	}
	return strings.TrimSuffix(s.String(), "\n")
}

func (r *RAM) Label() string {
	return r.label
}

func (r *RAM) Access(write bool, idx uint16, data uint8) (uint8, error) {
	if write {
		r.data[idx] = data
		return data, nil
	}
	return r.data[idx], nil
}
