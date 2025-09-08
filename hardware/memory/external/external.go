package external

import (
	"fmt"

	"github.com/jetsetilly/test7800/coprocessor"
	"github.com/jetsetilly/test7800/hardware/memory/external/elf"
)

type cartridge interface {
	Label() string
	Access(write bool, address uint16, data uint8) (uint8, error)
}

type Device struct {
	ctx      Context
	inserted cartridge
}

type Context interface {
	elf.Context
	Rand8Bit() uint8
}

func Create(ctx Context) *Device {
	dev := &Device{
		ctx: ctx,
	}
	return dev
}

func (dev *Device) Insert(c CartridgeInsertor) error {
	if c.creator == nil {
		dev.Eject()
		return nil
	}

	var err error
	dev.inserted, err = c.creator(dev.ctx, c.data)
	if err != nil {
		dev.Eject()
		return err
	}

	return nil
}

func (dev *Device) Eject() {
	dev.inserted = nil
}

func (dev *Device) IsEjected() bool {
	return dev.inserted == nil
}

func (dev *Device) Label() string {
	if dev.IsEjected() {
		return "Ejected"
	}
	return dev.inserted.Label()
}

func (dev *Device) Access(write bool, address uint16, data uint8) (uint8, error) {
	if dev.IsEjected() {
		return dev.ctx.Rand8Bit(), nil
	}

	v, err := dev.inserted.Access(write, address, data)
	if err != nil {
		return 0, fmt.Errorf("external: %s", err)
	}

	return v, nil
}

// external devices that are sensitive to changes in the address and data buses
// of the console will implement this interface
type busChangeSensitive interface {
	BusChange(address uint16, data uint8) error
}

func (dev *Device) BusChange(address uint16, data uint8) error {
	if d, ok := dev.inserted.(busChangeSensitive); ok {
		d.BusChange(address, data)
	}
	return nil
}

func (dev *Device) GetCoProcHandler() coprocessor.CartCoProcHandler {
	if d, ok := dev.inserted.(coprocessor.CartCoProcHandler); ok {
		return d
	}
	return nil
}

// external devices that want to know about the HLT line will implement the hlt interface
type hlt interface {
	HLT(bool)
}

// HLT should be called whenever the HLT line is changed
func (dev *Device) HLT(halt bool) {
	if d, ok := dev.inserted.(hlt); ok {
		d.HLT(halt)
	}
}
