package tia

type TIA struct {
}

func (tia *TIA) Label() string {
	return "TIA"
}

func (tia *TIA) Status() string {
	return tia.Label()
}

func (tia *TIA) Read(address uint16) (uint8, error) {
	return 0, nil
}

func (tia *TIA) Write(address uint16, data uint8) error {
	return nil
}
