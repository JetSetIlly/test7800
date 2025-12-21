package peripherals

type RIOT interface {
	PortWrite(idx uint16, data uint8, mask uint8) error
	Read(idx uint16) (uint8, error)
}

type TIA interface {
	PortWrite(idx uint16, data uint8, mask uint8) error
}
