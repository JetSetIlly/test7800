package hardware

import (
	"fmt"

	"github.com/jetsetilly/test7800/hardware/bios"
	"github.com/jetsetilly/test7800/hardware/cartridge"
	"github.com/jetsetilly/test7800/hardware/inptctrl"
	"github.com/jetsetilly/test7800/hardware/maria"
	"github.com/jetsetilly/test7800/hardware/ram"
	"github.com/jetsetilly/test7800/hardware/riot"
	"github.com/jetsetilly/test7800/hardware/tia"
)

type lastArea interface {
	Label() string
	Status() string
}

type memory struct {
	bios      *bios.BIOS
	INPTCTRL  *inptctrl.INPTCTRL
	MARIA     *maria.Maria
	RAM7800   *ram.RAM
	RAMRIOT   *ram.RAM
	TIA       *tia.TIA
	RIOT      *riot.RIOT
	cartridge *cartridge.Cartridge
	last      lastArea
}

func createMemory() *memory {
	mem := &memory{
		bios:      &bios.BIOS{},
		INPTCTRL:  &inptctrl.INPTCTRL{},
		MARIA:     &maria.Maria{},
		RAM7800:   ram.Create("ram7800", 0x1000),
		RAMRIOT:   ram.Create("ramRIOT", 0x0080),
		TIA:       &tia.TIA{},
		RIOT:      &riot.RIOT{},
		cartridge: &cartridge.Cartridge{},
	}
	return mem
}

type Area interface {
	// read and write both take an index value. this is an address in the area
	// but with the area origin removed. in other words, the area doesn't need
	// to know about it's location in memory, only the relative placement of
	// addresses within the area
	Read(idx uint16) (uint8, error)
	Write(idx uint16, data uint8) error
	Label() string
}

const (
	maskReadTIA                   = 0x000f
	maskWriteTIA                  = 0x003f
	maskReadRIOT                  = 0x00297
	maskWriteRIOT                 = 0x00287
	maskReadRIOT_timer            = uint16(0x284)
	maskReadRIOT_timer_correction = uint16(0x285)
)

// MapAddress returns the memory "area" and index into the area corresponding
// to the address.
//
// The result partially depends on the state of INPTCTRL. It will always return
// INPTCTRL, MARIA, etc. unless the Lock() is true and TIA() is true, in which
// case it may return TIA, INPTCTRL or MARIA depending on the address.
//
// It is possible for a nil Area to be returned. In which case, the index value
// will be zero.
//
// Also, RAM7800 is always returned as an area even if MARIA is disabled. I'm
// pretty sure this isn't strictly correct but it shouldn't cause any harm.
func (mem *memory) MapAddress(address uint16, read bool) (uint16, Area) {
	// page one
	if address >= 0x0000 && address <= 0x001f {
		// INPTCTRL or TIA
		if mem.INPTCTRL.Lock() {
			if read {
				return address & maskReadTIA, mem.TIA
			} else {
				return address & maskWriteTIA, mem.TIA
			}
		}
		return address, mem.INPTCTRL
	}
	if address >= 0x0020 && address <= 0x003f {
		// MARIA or TIA
		if mem.INPTCTRL.Lock() && mem.INPTCTRL.TIA() {
			if read {
				return address & maskReadTIA, mem.TIA
			} else {
				return address & maskWriteTIA, mem.TIA
			}
		}
		return address, mem.MARIA
	}
	if address >= 0x0040 && address <= 0x00ff {
		// RAM 7800 block 0
		return address - 0x0040 + 0x0840, mem.RAM7800
	}

	// page 2
	if address >= 0x0100 && address <= 0x011f {
		// INPTCTRL or TIA
		address -= 0x01000
		if mem.INPTCTRL.Lock() {
			if read {
				return address & maskReadTIA, mem.TIA
			} else {
				return address & maskWriteTIA, mem.TIA
			}
		}
		return address, mem.INPTCTRL
	}
	if address >= 0x0120 && address <= 0x013f {
		// MARIA or TIA
		address -= 0x0120
		if mem.INPTCTRL.Lock() && mem.INPTCTRL.TIA() {
			if read {
				return address & maskReadTIA, mem.TIA
			} else {
				return address & maskWriteTIA, mem.TIA
			}
		}
		return address, mem.MARIA
	}
	if address >= 0x0140 && address <= 0x01ff {
		// RAM 7800 block 1
		return address - 0x0140 + 0x0940, mem.RAM7800
	}

	// unsure

	if address >= 0x0280 && address <= 0x02ff {
		// RIOT
		address -= 0x0280
		if read {
			if address&maskReadRIOT_timer == maskReadRIOT_timer {
				return address & maskReadRIOT_timer_correction, mem.RIOT

			} else {
				return address & maskReadTIA, mem.RIOT
			}
		} else {
			return address & maskWriteRIOT, mem.RIOT
		}
	}

	// unsure

	if address >= 0x0380 && address <= 0x03ff {
		// RIOT
		address -= 0x0380
		if read {
			if address&maskReadRIOT_timer == maskReadRIOT_timer {
				return address & maskReadRIOT_timer_correction, mem.RIOT

			} else {
				return address & maskReadTIA, mem.RIOT
			}
		} else {
			return address & maskWriteRIOT, mem.RIOT
		}
	}

	// unsure

	if address >= 0x0480 && address <= 0x04ff {
		// RAM RIOT
		return address - 0x0480, mem.RAMRIOT
	}

	// unsure

	if address >= 0x1800 && address <= 0x27ff {
		// RAM 7800
		return address - 0x1800, mem.RAM7800
	}

	// BIOS
	if mem.INPTCTRL.BIOS() {
		if address >= bios.OriginBIOS && address <= 0xffff {
			return address - bios.OriginBIOS, mem.bios
		}
	}

	// cartridge
	if address >= cartridge.OriginCart && address <= 0xffff {
		return address - cartridge.OriginCart, mem.cartridge
	}

	return 0, nil
}

func (mem *memory) Read(address uint16) (uint8, error) {
	idx, area := mem.MapAddress(address, true)
	if area == nil {
		return 0, fmt.Errorf("memory.Read: unmapped address: %04x", address)
	}
	return area.Read(idx)
}

func (mem *memory) Write(address uint16, data uint8) error {
	idx, area := mem.MapAddress(address, false)
	if area == nil {
		return fmt.Errorf("memory.Write: unmapped address: %04x", address)
	}
	if l, ok := area.(lastArea); ok {
		mem.last = l
	}
	return area.Write(idx, data)
}
