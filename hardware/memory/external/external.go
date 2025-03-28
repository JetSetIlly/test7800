package external

type Device struct {
	ctx    Context
	data   []byte
	origin uint16
}

type Context interface {
	Rand8Bit() uint8
}

func Create(ctx Context) *Device {
	dev := &Device{
		ctx: ctx,
	}
	dev.Eject()
	return dev
}

func (dev *Device) Insert(d []byte) error {
	dev.data = d
	dev.origin = uint16(0x10000 - len(dev.data))
	return nil
}

func (dev *Device) Eject() {
	dev.data = dev.data[:0]
	dev.origin = 0xffff
}

func (dev *Device) IsEjected() bool {
	return len(dev.data) == 0
}

func (dev *Device) Label() string {
	if dev.IsEjected() {
		return "Ejected"
	}
	return "Cartridge"
}

func (dev *Device) Read(address uint16) (uint8, error) {
	if dev.IsEjected() {
		return dev.ctx.Rand8Bit(), nil
	}
	if address < dev.origin {
		return 0, nil
	}
	return dev.data[address-dev.origin], nil
}

func (dev *Device) Write(_ uint16, data uint8) error {
	return nil
}
