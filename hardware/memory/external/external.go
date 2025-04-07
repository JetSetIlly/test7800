package external

import (
	"bytes"
	"fmt"

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

func (dev *Device) Insert(d []byte) error {
	var err error

	// basic fingerprint
	if bytes.Contains(d, []byte{0x7f, 'E', 'L', 'F'}) {
		dev.inserted, err = elf.NewElf(dev.ctx, d)
	} else {
		dev.inserted, err = NewStandard(dev.ctx, d)
	}

	if err != nil {
		return fmt.Errorf("external: %s", err)
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
	return fmt.Sprintf("External: %s", dev.inserted.Label())
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
