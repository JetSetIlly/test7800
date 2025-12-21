package external

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

	// the type of controller to use for this cartridge
	Controller string

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
