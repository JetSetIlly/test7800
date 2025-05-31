package external

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/jetsetilly/test7800/hardware/memory/external/elf"
	"github.com/jetsetilly/test7800/logger"
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
	// https://forums.atariage.com/topic/333208-old-world-a78-format-10-31-primer/
	if bytes.Compare(d[0x01:0x0a], []byte("ATARI7800")) == 0 {
		logger.Logf(logger.Allow, "a78", "version %#02x", d[0x00])
		logger.Logf(logger.Allow, "a78", "%s", strings.TrimSpace(string(d[0x11:0x31])))

		size := (uint32(d[0x31]) << 24) | (uint32(d[0x32]) << 16) | (uint32(d[0x33]) << 8) | uint32(d[0x34])
		if len(d)-0x80 != int(size) {
			logger.Logf(logger.Allow, "a78", "cropping payload data to %d", size)
			d = d[:0x80+size]
		}

		cartType := (uint16(d[0x35]) << 8) | uint16(d[0x36])

		if cartType&0x40 == 0x40 {
			logger.Logf(logger.Allow, "a78", "POKEY required but not supported")
		}

		if cartType == 0x00 {
			return CartridgeInsertor{
				data: d,
				creator: func(ctx Context, d []uint8) (cartridge, error) {
					return NewFlat(ctx, d[0x80:])
				},
			}, nil
		} else {
			if cartType&0x02 == 0x02 {
				return CartridgeInsertor{
					data: d,
					creator: func(ctx Context, d []uint8) (cartridge, error) {
						return NewSuper(ctx, d[0x80:],
							cartType&0x08 == 0x08,
							cartType&0x04 == 0x04)
					},
				}, nil
			} else {
				return CartridgeInsertor{}, fmt.Errorf("unsupported a78 cartridge type (%#02x)", cartType)
			}
		}
	}

	// check to see if data contains any non-ASCII bytes. if it does then we assume
	// it is a flat cartridge dump. data continaing only ASCII suggests that it is a
	// script or a boot file that can be further interpreted by the debugger, but we
	// don't worry about that here
	for _, c := range d {
		if c > unicode.MaxASCII {
			return CartridgeInsertor{
				data: d,
				creator: func(ctx Context, d []uint8) (cartridge, error) {
					return NewFlat(ctx, d[:])
				},
			}, nil
		}
	}

	return CartridgeInsertor{}, UnrecognisedData
}
