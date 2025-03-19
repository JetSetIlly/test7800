package cartridge

const OriginCart = 0x3000

type Cartridge struct {
}

func (cart *Cartridge) Ejected() bool {
	return true
}

func (cart *Cartridge) Label() string {
	return "Cartridge"
}

func (cart *Cartridge) Read(idx uint16) (uint8, error) {
	return 0, nil
}

func (cart *Cartridge) Write(_ uint16, data uint8) error {
	return nil
}
