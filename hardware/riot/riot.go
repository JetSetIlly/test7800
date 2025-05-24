package riot

import (
	"fmt"
	"slices"
)

// TODO: SWACNT and SWBCNT

type RIOT struct {
	mem Memory

	swcha uint8
	swchb uint8

	// selected timer
	divider int

	// timer output
	intim  uint8
	timint uint8

	// ticksRemaining is the number of CPU cycles remaining before the
	// value is decreased. the following rules apply:
	//		* set to 0 when new timer is set
	//		* causes value to decrease whenever it reaches -1
	//		* is reset to divider whenever value is decreased
	// with regards to the last point, note that divider changes to 1
	// once INTIMvalue reaches 0
	ticksRemaining int

	// the index of the last RIOT address to be read. this only affects what happens
	// to the timer under certain conditions so in practice it's only ever set
	// when INTIM or TIMINT is read, so it's only ever set to 0x04 or 0x05
	//
	// set to 0 if the last read was not to the RIOT
	lastReadIdx int
}

const (
	timintExpired = uint8(0x80)
	timintPA7     = uint8(0x40)
)

type Memory interface {
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
}

func Create(mem Memory) *RIOT {
	riot := &RIOT{
		mem: mem,
	}
	riot.Reset()
	return riot
}

func (riot *RIOT) Reset() {
	riot.swcha = 0xff
	riot.swchb = 0xff
	riot.timint = timintPA7
	riot.lastReadIdx = 0
	riot.setTimer(1024, 0)
}

func (riot *RIOT) Label() string {
	return "RIOT"
}

func (riot *RIOT) Status() string {
	return riot.Label()
}

func (riot *RIOT) Access(write bool, idx uint16, data uint8) (uint8, error) {
	if write {
		return data, riot.Write(idx, data)
	}
	return riot.Read(idx)
}

func (riot *RIOT) Read(idx uint16) (uint8, error) {
	switch idx {
	case 0x00:
		return riot.swcha, nil
	case 0x01:
		// SWACNT
		return 0, nil
	case 0x02:
		return riot.swchb, nil
	case 0x03:
		// SWBCNT
		return 0, nil
	case 0x04:
		riot.lastReadIdx = 0x04
		return riot.intim, nil
	case 0x05:
		riot.lastReadIdx = 0x05
		return riot.timint, nil
	}
	return 0, nil
}

func (riot *RIOT) Poke(idx uint16, data uint8) error {
	return riot.Write(idx, data)
}

func (riot *RIOT) Write(idx uint16, data uint8) error {
	switch idx {
	case 0x00:
		riot.swcha = data
	case 0x01:
		// SWACNT
	case 0x02:
		riot.swchb = data
	case 0x03:
		// SWBCNT
	case 0x04, 0x14:
		riot.setTimer(1, data)
	case 0x05, 0x15:
		riot.setTimer(8, data)
	case 0x06, 0x16:
		riot.setTimer(64, data)
	case 0x07, 0x17:
		riot.setTimer(1024, data)
	}
	return nil
}

func (riot *RIOT) setTimer(divider int, data uint8) {
	if !slices.Contains([]int{1, 8, 64, 1024}, divider) {
		panic(fmt.Errorf("%d is not a valid RIOT timer divider", divider))
	}

	riot.divider = divider

	// writing to INTIM register has a similar effect on the expired bit of the
	// TIMINT register as reading. See commentary in the Tick() function
	if riot.ticksRemaining == 0 && riot.intim == 0xff {
		riot.timint |= timintExpired
	} else {
		riot.timint &= ^timintExpired
	}

	// the ticks remaining value should be zero or one for accurate timing (as
	// tested with these test ROMs https://github.com/stella-emu/stella/issues/108)
	//
	// I'm not sure which value is correct so setting at zero until there's a
	// good reason to do otherwise
	//
	// note however, the internal values in the emulated machine (and as reported by
	// the debugger) will not match the debugging values in stella. to match
	// the debugging values in stella a value of 2 is required
	riot.ticksRemaining = 0

	// write value to INTIM straight-away
	riot.intim = data
}

func (riot *RIOT) Tick() {
	switch riot.lastReadIdx {
	case 0x04:
		// if INTIM is *read* then the decrement reverts to once per timer
		// divider. this won't have any discernable effect unless the timer
		// divider has been flipped to 1 when INTIM cycles back to 255
		//
		// if the expired flag has *just* been set (ie. in the previous cycle)
		// then do not do the reversion. see discussion:
		//
		// https://atariage.com/forums/topic/303277-to-roll-or-not-to-roll/
		//
		// https://atariage.com/forums/topic/133686-please-explain-riot-timmers/?do=findComment&comment=1617207
		if riot.ticksRemaining != 0 || riot.intim != 0xff {
			riot.timint &= ^timintExpired
		}
	case 0x05:
		// from the NMOS 6532:
		//
		// "Clearing of the PA7 Interrupt Flag occurs when the microprocessor
		// reads the Interrupt Flag Register."
		//
		// and from the Rockwell 6532 documentation:
		//
		// "To clear PA7 interrupt flag, simply read the Interrupt Flag
		// Register"

		// update PA7 bit and TIMINT value if necessary. writing the TIMINT
		// value is necessary because the PA7 bit has changed
		//
		// a previous version of the emulator didn't do this meaning that
		// the PA7 bit in the register was updated only once the timer had
		// expired (see below)
		riot.timint &= ^timintPA7
	}
	riot.lastReadIdx = 0

	riot.ticksRemaining--
	if riot.ticksRemaining <= 0 {
		riot.intim--
		if riot.intim == 0xff {
			riot.timint |= timintExpired
		}

		if riot.timint&timintExpired == timintExpired {
			riot.ticksRemaining = 0
		} else {
			riot.ticksRemaining = int(riot.divider)
		}
	}
}
