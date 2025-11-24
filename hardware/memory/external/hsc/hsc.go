package hsc

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/jetsetilly/test7800/hardware/spec"
	"github.com/jetsetilly/test7800/logger"
	"github.com/jetsetilly/test7800/resources"
)

//go:embed "hsc_pal.bin"
var pal []byte

//go:embed "hsc.bin"
var ntsc []byte

//go:embed "initialised.bin"
var initialised []byte

const (
	biosOrigin = 0x3000
	biosMemtop = 0x3fff
	sramOrigin = 0x1000
	sramMemtop = 0x17ff
)

func init() {
	if len(pal) != biosMemtop-biosOrigin+1 {
		panic("HSC PAL bios is incorrect length. should be 4096 bytes")
	}
	if len(ntsc) != biosMemtop-biosOrigin+1 {
		panic("HSC NTSC bios is incorrect length. should be 4096 bytes")
	}
	if len(initialised) != sramMemtop-sramOrigin+1 {
		panic("initial HSC data is incorrect length. should be 2048 bytes")
	}
}

type Context interface {
	Spec() spec.Spec
}

type Bus interface {
	Label() string
	Access(write bool, address uint16, data uint8) (uint8, error)
}

type Device struct {
	ctx      Context
	inserted Bus
	bios     []uint8
	sram     []uint8
}

// the resource path to the nvram files
const hsc_nvram = "hsc_nvram"

func Create(ctx Context, cartridge Bus) *Device {
	dev := &Device{
		ctx:      ctx,
		inserted: cartridge,
	}

	switch ctx.Spec().ID {
	case "PAL":
		dev.bios = pal[:]
	default:
		dev.bios = ntsc[:]
	}

	dev.restore()
	if dev.sram == nil {
		dev.sram = initialised[:]
	}
	dev.save()

	return dev
}

func (dev *Device) Label() string {
	if dev.inserted != nil {
		return fmt.Sprintf("%s [via HSC]", dev.inserted.Label())
	}
	return "no cartridge [via HSC]"
}

func (dev *Device) Access(write bool, address uint16, data uint8) (uint8, error) {
	if address >= biosOrigin && address <= biosMemtop {
		if write {
			return 0, nil
		}
		return dev.bios[address-biosOrigin], nil
	}
	if address >= sramOrigin && address <= sramMemtop {
		if write {
			dev.sram[address-sramOrigin] = data
			dev.save()
			return 0, nil
		}
		return dev.sram[address-sramOrigin], nil
	}

	if dev.inserted != nil {
		return dev.inserted.Access(write, address, data)
	}
	return 0, nil
}

func (dev *Device) save() {
	p, err := resources.JoinPath(hsc_nvram)
	if err != nil {
		logger.Log(logger.Allow, "HSC", err)
		return
	}

	f, err := os.Create(p)
	if err != nil {
		logger.Log(logger.Allow, "HSC", err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Log(logger.Allow, "HSC", err)
		}
	}()

	f.Write(dev.sram)
}

func (dev *Device) restore() {
	p, err := resources.JoinPath(hsc_nvram)
	if err != nil {
		logger.Log(logger.Allow, "HSC", err)
		return
	}

	st, err := os.Stat(p)
	if err != nil {
		logger.Log(logger.Allow, "HSC", err)
		return
	}

	sz := int64(sramMemtop - sramOrigin + 1)
	if st.Size() != sz {
		logger.Logf(logger.Allow, "HSC", "%s is not %d bytes in size", p, sz)
		return
	}

	f, err := os.Open(p)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Log(logger.Allow, "HSC", err)
		}
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Log(logger.Allow, "HSC", err)
		}
	}()

	dev.sram = make([]byte, sz)
	f.Read(dev.sram)
}
