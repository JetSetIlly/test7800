package external

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"unicode"

	"github.com/jetsetilly/test7800/hardware/memory/external/elf"
	"github.com/jetsetilly/test7800/hardware/pokey"
	"github.com/jetsetilly/test7800/logger"
)

// CartridgeReset contains 'instructions' to be followed when the cartridge is inserted
type CartridgeReset struct {
	// if BypassBIOS is true then the normal BIOS initialisation procedure is bypassed
	BypassBIOS bool
}

type CartridgeInsertor struct {
	filename string
	data     []uint8

	// returns a new instance of the cartridge. this will be the
	creator func(Context, []uint8) (Bus, error)

	// returns the actions to take on cartridge reset
	reset CartridgeReset

	// whether controller should have just one-buttons. NOTE: placeholder
	// until we add more sophisticated controller requirements (paddle, etc.)
	OneButtonStick bool

	// tv specifiction. if the string is empty then the spec of the console is not changed
	spec string

	// list of additional chips (eg. POKEYs) that are present in the cartridge
	chips []func(Context) (OptionalBus, error)

	// use high-score cartridge shim with cartridge
	UseHSC     bool
	UseSavekey bool
}

func (c CartridgeInsertor) Filename() string {
	return c.filename
}

func (c CartridgeInsertor) Data() []uint8 {
	return c.data
}

func (c CartridgeInsertor) Spec() string {
	return c.spec
}

func (c CartridgeInsertor) ResetProcedure() CartridgeReset {
	return c.reset
}

// error returned when data is not recognised at all
var UnrecognisedData = errors.New("unrecognised data")

func Fingerprint(filename string, mapper string) (CartridgeInsertor, error) {
	d, err := os.ReadFile(filename)
	if err != nil {
		return CartridgeInsertor{}, err
	}
	return FingerprintBlob(filename, d, mapper)
}

func FingerprintBlob(filename string, d []uint8, mapper string) (CartridgeInsertor, error) {
	// normalise mapper string
	mapper = strings.ToUpper(mapper)

	// try ELF first because it's the most solidly defined of all ROM types
	if slices.Contains([]string{"ELF", "AUTO"}, mapper) {
		if bytes.Contains(d, []byte{0x7f, 'E', 'L', 'F'}) {
			return CartridgeInsertor{
				data: d,
				creator: func(ctx Context, d []uint8) (Bus, error) {
					return elf.NewElf(ctx, d)
				},
				reset: CartridgeReset{
					BypassBIOS: true,
				},
			}, nil
		}

		// if user requested ELF explicitely as the mapper then return an error
		if mapper == "ELF" {
			return CartridgeInsertor{}, fmt.Errorf("file is not an ELF ROM")
		}
	}

	// a78 header
	// https://7800.8bitdev.org/index.php/A78_Header_Specification
	// https://forums.atariage.com/topic/333208-old-world-a78-format-10-31-primer/
	if slices.Contains([]string{"A78", "AUTO"}, mapper) {
		if bytes.Equal(d[0x01:0x0a], []byte("ATARI7800")) {
			version := d[0x00]

			// log a78 version and game title
			logger.Logf(logger.Allow, "a78", "version: %#02x", version)
			logger.Logf(logger.Allow, "a78", "title: %s", strings.TrimSpace(string(d[0x11:0x31])))

			const endOfHeader = "ACTUAL CART DATA STARTS HERE"
			dataStart := bytes.Index(d, []uint8(endOfHeader))
			if dataStart == -1 {
				return CartridgeInsertor{}, fmt.Errorf("malfored A78 header. no end of header indicator")
			}
			dataStart += len(endOfHeader)

			// cartridge size
			size := (uint32(d[0x31]) << 24) | (uint32(d[0x32]) << 16) | (uint32(d[0x33]) << 8) | uint32(d[0x34])
			if len(d)-dataStart != int(size) {
				logger.Logf(logger.Allow, "a78", "cropping payload data to %d", size)
				d = d[:dataStart+int(size)]
			}

			// controller type
			var oneButtonStick bool
			controllerP0 := d[0x37]
			switch controllerP0 {
			case 0x00:
				// no controller, don't care
			case 0x01:
				oneButtonStick = false
				logger.Logf(logger.Allow, "a78", "controllers: using two-button stick")
			case 0x05:
				oneButtonStick = true
				logger.Logf(logger.Allow, "a78", "controllers: using one-button stick")
			case 0x0b:
				oneButtonStick = false
				logger.Log(logger.Allow, "a78", "controllers: SNES2Atari emulated as two-button stick")
			}

			// tv spec
			var spec string
			if d[0x39]&0x01 == 0x01 {
				spec = "PAL"
			} else {
				spec = "NTSC"
			}

			// save device
			useHSC := d[0x3a]&0x01 == 0x01
			useSavekey := d[0x3a]&0x02 == 0x02

			// cartridge type
			cartType := (uint16(d[0x35]) << 8) | uint16(d[0x36])
			logger.Logf(logger.Allow, "a78", "cart type: %08b %08b", uint8(cartType>>8), uint8(cartType))

			if cartType&0x0800 == 0x0800 {
				logger.Logf(logger.Allow, "a78", "YM2151 required but not supported")
				cartType &= (0x0800 ^ 0xffff)
			}

			// list of creator functions for additional chips
			var chips []func(Context) (OptionalBus, error)

			if cartType&0x0001 == 0x0001 {
				pk := func(ctx Context) (OptionalBus, error) {
					return pokey.NewAudio(ctx, 0x4000)
				}
				chips = append(chips, pk)
				cartType &= (0x0001 ^ 0xffff)
			}
			if cartType&0x0040 == 0x0040 {
				pk := func(ctx Context) (OptionalBus, error) {
					return pokey.NewAudio(ctx, 0x0450)
				}
				chips = append(chips, pk)
				cartType &= (0x0040 ^ 0xffff)
			}
			if cartType&0x0400 == 0x0400 {
				pk := func(ctx Context) (OptionalBus, error) {
					return pokey.NewAudio(ctx, 0x0440)
				}
				chips = append(chips, pk)
				cartType &= (0x0400 ^ 0xffff)
			}
			if cartType&0x8000 == 0x8000 {
				pk := func(ctx Context) (OptionalBus, error) {
					return pokey.NewAudio(ctx, 0x0800)
				}
				chips = append(chips, pk)
				cartType &= (0x8000 ^ 0xffff)
			}

			if cartType == 0x0000 {
				return CartridgeInsertor{
					data: d,
					creator: func(ctx Context, d []uint8) (Bus, error) {
						return NewFlat(ctx, d[dataStart:])
					},
					OneButtonStick: oneButtonStick,
					spec:           spec,
					chips:          chips,
					UseHSC:         useHSC,
					UseSavekey:     useSavekey,
				}, nil
			}

			// activision
			if cartType == 0x0100 {
				// if cartridge name contians the '(OM)' string then the cartridge has been dumped
				// with "original ordering". alternative ordering can be indicated with '(AM)' but
				// we don't look for that and we assume that type of ordering by default
				originalOrder := strings.Contains(filename, "(OM)")

				return CartridgeInsertor{
					filename: filename,
					data:     d,
					creator: func(ctx Context, d []uint8) (Bus, error) {
						return NewActivision(ctx, d[dataStart:], originalOrder)
					},
					OneButtonStick: oneButtonStick,
					spec:           spec,
					chips:          chips,
					UseHSC:         useHSC,
					UseSavekey:     useSavekey,
				}, nil
			}

			// absolute
			if cartType == 0x0200 {
				return CartridgeInsertor{
					filename: filename,
					data:     d,
					creator: func(ctx Context, d []uint8) (Bus, error) {
						return NewAbsolute(ctx, d[dataStart:])
					},
					OneButtonStick: oneButtonStick,
					spec:           spec,
					chips:          chips,
					UseHSC:         useHSC,
					UseSavekey:     useSavekey,
				}, nil
			}

			// banksets
			if cartType&0x2000 == 0x2000 {
				supergame := cartType&0x02 == 0x02
				banksetRAM := cartType&0x4000 == 0x4000
				return CartridgeInsertor{
					filename: filename,
					data:     d,
					creator: func(ctx Context, d []uint8) (Bus, error) {
						return NewBanksets(ctx, supergame, d[dataStart:], banksetRAM)
					},
					OneButtonStick: oneButtonStick,
					spec:           spec,
					chips:          chips,
					UseHSC:         useHSC,
					UseSavekey:     useSavekey,
				}, nil
			}

			// supergame
			banked := cartType&0x02 == 0x02
			exram := cartType&0x04 == 0x04
			exrom := cartType&0x08 == 0x08

			if banked || exrom || exram {
				return CartridgeInsertor{
					filename: filename,
					data:     d,
					creator: func(ctx Context, d []uint8) (Bus, error) {
						return NewSupergame(ctx, d[dataStart:],
							banked, exram, exrom,
						)
					},
					OneButtonStick: oneButtonStick,
					spec:           spec,
					chips:          chips,
					UseHSC:         useHSC,
					UseSavekey:     useSavekey,
				}, nil
			}

			return CartridgeInsertor{}, fmt.Errorf("a78: unsupported cartridge type (%#04x)", cartType)
		}

		// if user requested A78 explicitely as the mapper then return an error
		if mapper == "A78" {
			return CartridgeInsertor{}, fmt.Errorf("file is not an A78 ROM")
		}
	}

	// SN/EAGLE mapper
	if slices.Contains([]string{"SN", "EAGLE"}, mapper) {
		return CartridgeInsertor{
			filename: filename,
			data:     d,
			creator: func(ctx Context, d []uint8) (Bus, error) {
				return NewSN(ctx, d[:], mapper)
			},
			OneButtonStick: false,
		}, nil
	}

	// check to see if data contains any non-ASCII bytes. if it does then we assume
	// it is a flat cartridge dump. data continaing only ASCII suggests that it is a
	// script or a boot file that can be further interpreted by the debugger
	for _, c := range d {
		if c > unicode.MaxASCII {
			return CartridgeInsertor{
				filename: filename,
				data:     d,
				creator: func(ctx Context, d []uint8) (Bus, error) {
					return NewFlat(ctx, d[:])
				},
				OneButtonStick: false,
			}, nil
		}
	}

	return CartridgeInsertor{
		filename: filename,
		data:     d,
	}, UnrecognisedData
}
