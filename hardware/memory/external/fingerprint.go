package external

import (
	"bytes"

	"github.com/jetsetilly/test7800/hardware/memory/external/elf"
)

type CartridgeInsertor struct {
	data    []uint8
	creator func(Context, []uint8) (cartridge, error)
}

func (c CartridgeInsertor) Valid() bool {
	return c.creator != nil
}

func Fingerprint(d []uint8) CartridgeInsertor {
	if bytes.Contains(d, []byte{0x7f, 'E', 'L', 'F'}) {
		return CartridgeInsertor{
			data: d,
			creator: func(ctx Context, d []uint8) (cartridge, error) {
				return elf.NewElf(ctx, d)
			},
		}
	}

	if bytes.Compare(d[1:10], []byte("ATARI7800")) == 0 {
		return CartridgeInsertor{
			data: d,
			creator: func(ctx Context, d []uint8) (cartridge, error) {
				return NewStandard(ctx, d)
			},
		}
	}

	return CartridgeInsertor{}
}
