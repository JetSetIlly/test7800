package ram

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

type RAM struct {
	label string
	data  []uint8
}

func Create(label string, size int) RAM {
	return RAM{
		label: label,
		data:  make([]uint8, size),
	}
}

func (r *RAM) Reset(random bool) {
	if random {
		for i := range len(r.data) {
			r.data[i] = uint8(rand.IntN(255))
		}
	} else {
		clear(r.data)
	}
}

func (r *RAM) String() string {
	var s strings.Builder
	for i := 0; i <= (len(r.data)-1)/16; i++ {
		j := i * 15
		s.WriteString(fmt.Sprintf("% 02x\n", r.data[j:j+15]))
	}
	return strings.TrimSuffix(s.String(), "\n")
}

func (r *RAM) Label() string {
	return r.label
}

func (r *RAM) Read(idx uint16) (uint8, error) {
	return r.data[idx], nil
}

func (r *RAM) Write(idx uint16, data uint8) error {
	r.data[idx] = data
	return nil
}
