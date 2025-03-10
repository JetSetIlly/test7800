package riot

type RIOT struct {
}

func (riot *RIOT) Label() string {
	return "RIOT"
}

func (riot *RIOT) Status() string {
	return riot.Label()
}

func (riot *RIOT) Read(address uint16) (uint8, error) {
	return 0, nil
}

func (riot *RIOT) Write(address uint16, data uint8) error {
	return nil
}
