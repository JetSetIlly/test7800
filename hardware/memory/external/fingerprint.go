package external

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/jetsetilly/test7800/hardware/memory/external/elf"
)

type CartridgeReset struct {
	// if BypassBIOS is true then the normal BIOS initialisation procedure is
	// bypassed. in this case the INPTCTRL field is important for setting the
	BypassBIOS bool
}

type CartridgeInsertor struct {
	data    []uint8
	creator func(Context, []uint8) (cartridge, error)
	reset   CartridgeReset
}

func (c CartridgeInsertor) ResetProcedure() CartridgeReset {
	return c.reset
}

// error returned when data is not recognised at all
var UnrecognisedData = errors.New("unrecognised data")

func Fingerprint(d []uint8) (CartridgeInsertor, error) {
	if bytes.Contains(d, []byte{0x7f, 'E', 'L', 'F'}) {
		return CartridgeInsertor{
			data: d,
			creator: func(ctx Context, d []uint8) (cartridge, error) {
				return elf.NewElf(ctx, d)
			},
			reset: CartridgeReset{
				BypassBIOS: true,
			},
		}, nil
	}

	// a78 header
	// https://7800.8bitdev.org/index.php/A78_Header_Specification
	if bytes.Compare(d[1:10], []byte("ATARI7800")) == 0 {
		title := strings.TrimSpace(string(d[17:49]))
		cartType := (uint16(d[53]) << 8) | uint16(d[54])

		_ = title

		if cartType == 0x00 {
			return CartridgeInsertor{
				data: d,
				creator: func(ctx Context, d []uint8) (cartridge, error) {
					return NewStandard(ctx, d[128:])
				},
			}, nil
		} else {
			return CartridgeInsertor{}, fmt.Errorf("unsupported a78 cartridge type (%#02x)", cartType)
		}
	}

	return CartridgeInsertor{}, UnrecognisedData
}
