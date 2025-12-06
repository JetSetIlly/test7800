package external

import (
	"fmt"

	"github.com/jetsetilly/test7800/coprocessor"
	"github.com/jetsetilly/test7800/hardware/memory/external/elf"
	"github.com/jetsetilly/test7800/logger"
)

type Bus interface {
	Label() string
	Access(write bool, address uint16, data uint8) (uint8, error)
}

// the OptionalBus interface differs to the Bus interface because the Access() function returns an
// additional boolean result to indicate whether the address was recognised and handled. The
// intention is for the External device to call Access() on the OptionalBus first and to only call
// Access() on the main Bus device if the first Access() returned false
type OptionalBus interface {
	Label() string
	Access(write bool, address uint16, data uint8) (uint8, bool, error)
}

type Device struct {
	ctx      Context
	inserted Bus
	chips    []OptionalBus
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
	// eject any existing device to make sure we forget about any chips on the optional bus
	dev.Eject()

	if c.creator == nil {
		return nil
	}

	var err error
	dev.inserted, err = c.creator(dev.ctx, c.data)
	if err != nil {
		dev.Eject()
		return err
	}

	for i := range c.chips {
		s, err := c.chips[i](dev.ctx)
		if err != nil {
			dev.Eject()
			return err
		}
		dev.chips = append(dev.chips, s)
		logger.Log(logger.Allow, "chips", s.Label())
	}

	return nil
}

func (dev *Device) Eject() {
	dev.inserted = nil
	dev.chips = dev.chips[:0]
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

	for i := range dev.chips {
		v, ok, err := dev.chips[i].Access(write, address, data)
		if err != nil {
			return 0, fmt.Errorf("external: %s", err)
		}
		if ok {
			return v, nil
		}
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

func (dev *Device) GetCoProcBus() coprocessor.CartCoProcBus {
	if d, ok := dev.inserted.(coprocessor.CartCoProcBus); ok {
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

// Chips iterates through the additional (none ROM/RAM) chips in the external device
func (dev *Device) Chips(yield func(OptionalBus)) {
	for _, c := range dev.chips {
		yield(c)
	}
}
