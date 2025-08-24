package external

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/jetsetilly/test7800/hardware/memory/external/elf"
	"github.com/jetsetilly/test7800/logger"
)

type CartridgeReset struct {
	// if BypassBIOS is true then the normal BIOS initialisation procedure is bypassed
	BypassBIOS bool
}

type CartridgeInsertor struct {
	filename string
	data     []uint8
	creator  func(Context, []uint8) (cartridge, error)
	reset    CartridgeReset

	// whether controller should have two-buttons. NOTE: placeholder
	// until we add more sophisticated controller requirements (paddle, etc.)
	TwoButtonStick bool
}

func (c CartridgeInsertor) Filename() string {
	return c.filename
}

func (c CartridgeInsertor) Data() []uint8 {
	return c.data
}

func (c CartridgeInsertor) ResetProcedure() CartridgeReset {
	return c.reset
}

// error returned when data is not recognised at all
var UnrecognisedData = errors.New("unrecognised data")

func Fingerprint(filename string) (CartridgeInsertor, error) {
	d, err := os.ReadFile(filename)
	if err != nil {
		return CartridgeInsertor{}, err
	}

	// only allow ELF loading if an "ALLOW_ELF" file is present in the current directory
	f, err := os.Open("ALLOW_ELF")
	if err == nil {
		defer f.Close()
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
	}

	// a78 header
	// https://7800.8bitdev.org/index.php/A78_Header_Specification
	// https://forums.atariage.com/topic/333208-old-world-a78-format-10-31-primer/
	if bytes.Equal(d[0x01:0x0a], []byte("ATARI7800")) {
		// log a78 version and game title
		logger.Logf(logger.Allow, "a78", "version: %#02x", d[0x00])
		logger.Logf(logger.Allow, "a78", "title: %s", strings.TrimSpace(string(d[0x11:0x31])))

		// cartridge size
		size := (uint32(d[0x31]) << 24) | (uint32(d[0x32]) << 16) | (uint32(d[0x33]) << 8) | uint32(d[0x34])
		if len(d)-0x80 != int(size) {
			logger.Logf(logger.Allow, "a78", "cropping payload data to %d", size)
			d = d[:0x80+size]
		}

		// controller type
		var twoButtonStick bool
		controllerP0 := d[0x37]
		switch controllerP0 {
		case 0x00:
			// no controller, don't care
		case 0x01:
			twoButtonStick = true
			logger.Logf(logger.Allow, "a78", "controllers: using two-button stick")
		case 0x05:
			twoButtonStick = false
			logger.Logf(logger.Allow, "a78", "controllers: using one-button stick")
		default:
			return CartridgeInsertor{}, fmt.Errorf("a78: unsupported controller (%#02x)", controllerP0)
		}

		// cartridge type
		cartType := (uint16(d[0x35]) << 8) | uint16(d[0x36])
		cartType_info := cartType
		logger.Logf(logger.Allow, "a78", "cart type: %08b %08b", uint8(cartType>>8), uint8(cartType))

		if cartType&0x0001 == 0x0001 {
			logger.Logf(logger.Allow, "a78", "POKEY (at $4000) required but not supported")
			cartType &= (0x01 ^ 0xff)
		}
		if cartType&0x0040 == 0x0040 {
			logger.Logf(logger.Allow, "a78", "POKEY (at $440) required but not supported")
			cartType &= (0x0040 ^ 0xff)
		}
		if cartType&0x0800 == 0x0800 {
			logger.Logf(logger.Allow, "a78", "YM2151 required but not supported")
			cartType &= (0x0800 ^ 0xff)
		}
		if cartType&0x8000 == 0x8000 {
			logger.Logf(logger.Allow, "a78", "POKEY (at $800) required but not supported")
			cartType &= (0x8000 ^ 0xff)
		}

		// flat cartridge type
		if cartType == 0x00 {
			return CartridgeInsertor{
				data: d,
				creator: func(ctx Context, d []uint8) (cartridge, error) {
					return NewFlat(ctx, d[0x80:])
				},
				TwoButtonStick: twoButtonStick,
			}, nil
		}

		// supergame bits
		banked := cartType&0x02 == 0x02
		exram := cartType&0x04 == 0x04
		exrom := cartType&0x08 == 0x08

		if banked || exrom || exram {
			return CartridgeInsertor{
				filename: filename,
				data:     d,
				creator: func(ctx Context, d []uint8) (cartridge, error) {
					return NewSupergame(ctx, d[0x80:],
						banked, exrom, exram,
					)
				},
				TwoButtonStick: twoButtonStick,
			}, nil
		}

		return CartridgeInsertor{}, fmt.Errorf("a78: unsupported cartridge type (%#04x)", cartType_info)
	}

	// check to see if data contains any non-ASCII bytes. if it does then we assume
	// it is a flat cartridge dump. data continaing only ASCII suggests that it is a
	// script or a boot file that can be further interpreted by the debugger
	for _, c := range d {
		if c > unicode.MaxASCII {
			return CartridgeInsertor{
				filename: filename,
				data:     d,
				creator: func(ctx Context, d []uint8) (cartridge, error) {
					return NewFlat(ctx, d[:])
				},

				// default to two button stick if we don't have a header
				TwoButtonStick: true,
			}, nil
		}
	}

	return CartridgeInsertor{
		filename: filename,
		data:     d,
	}, UnrecognisedData
}
