package memory

import (
	"fmt"

	"github.com/jetsetilly/test7800/hardware/memory/bios"
	"github.com/jetsetilly/test7800/hardware/memory/cartridge"
	"github.com/jetsetilly/test7800/hardware/memory/inptctrl"
	"github.com/jetsetilly/test7800/hardware/memory/ram"
)

type Memory struct {
	BIOS      *bios.BIOS
	INPTCTRL  *inptctrl.INPTCTRL
	RAM7800   *ram.RAM
	RAMRIOT   *ram.RAM
	MARIA     Area
	TIA       Area
	RIOT      Area
	cartridge *cartridge.Cartridge
	Last      Area
}

type Context interface {
	ram.Context
	cartridge.Context
}

func Create(ctx Context) (*Memory, AddChips) {
	mem := &Memory{
		BIOS:      &bios.BIOS{},
		INPTCTRL:  &inptctrl.INPTCTRL{},
		RAM7800:   ram.Create(ctx, "ram7800", 0x1000),
		RAMRIOT:   ram.Create(ctx, "ramRIOT", 0x0080),
		cartridge: cartridge.Create(ctx),
	}
	return mem, func(maria Area, tia Area, riot Area) {
		mem.MARIA = maria
		mem.TIA = tia
		mem.RIOT = riot
	}
}

func (mem *Memory) Reset(random bool) {
	mem.INPTCTRL.Reset()
	mem.RAM7800.Reset(random)
	mem.RAMRIOT.Reset(random)
}

// AddChips is returned by the Create() function and should be called to
// finalise the memory creation process
type AddChips func(maria Area, tia Area, riot Area)

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
func (mem *Memory) MapAddress(address uint16, read bool) (uint16, Area) {
	// map taken from "7800 Software Guide":
	//
	// 0000 to 001F 	TIA Registers
	// 0020 to 003F 	MARIA Registers
	// 0040 to 00FF 	RAM (6116 Block Zero)
	// 0100 to 013F 	Shadow of Page 0
	// 0140 to 01FF 	RAM (6116 Block One)
	// 0200 to 027F 	Shadowed
	// 0280 to 02FF 	6532 Ports
	// 0300 to 037F 	Shadowed
	// 0380 to 03FF 	Shadowed 6532 Ports
	// 0400 to 047F 	Available for mapping by external devices
	// 0480 to 04FF 	6532 RAM. Don't Use
	// 0500 to 057F 	Available for mapping by external devices
	// 0580 to 05FF 	6532 RAM Shadow. Don't Use
	// 0600 to 17FF 	Available for mapping by external devices
	// 1800 to 203F 	RAM
	// 2040 to 20FF 	Block Zero Shadow
	// 2100 to 213F 	RAM
	// 2140 to 21FF 	Block One Shadow
	// 2200 to 27FF 	RAM
	// 2800 to 2FFF 	Unavailable for mapping by external devices. (BIOS conflict)
	// 3000 to FF7F 	Available for mapping by external devices
	// FF80 to FFF7 	Reserved for cart encryption signature
	// FFF8 to FFFF 	Reserved for startup flags and 6502 vectors
	//
	// the range 1800 to 27ff is treated as a single block of RAM. contrary to
	// what the map says, I think that the areas referred to as "SHADOW OF ZERO
	// PAGE RAM" and "SHADOW OF STACK RAM" are better thought of as the primary
	// areas and "ZERO PAGE RAM" and "RAM (STACK)" as being the shadows
	//
	// The MARIA.S file for the 7800 PAL OS source code shows a slightly
	// different map to the above. we prefer this software guide map because it
	// is based on modern research

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
		address -= 0x0100
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
		address -= 0x0100
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
		// RAM 7800 block 1 (6507 stack)
		return address - 0x0140 + 0x0940, mem.RAM7800
	}

	// page 3
	if address >= 0x0200 && address <= 0x021f {
		// INPTCTRL or TIA
		address -= 0x0200
		if mem.INPTCTRL.Lock() {
			if read {
				return address & maskReadTIA, mem.TIA
			} else {
				return address & maskWriteTIA, mem.TIA
			}
		}
		return address, mem.INPTCTRL
	}
	if address >= 0x0220 && address <= 0x023f {
		// MARIA or TIA
		address -= 0x0200
		if mem.INPTCTRL.Lock() && mem.INPTCTRL.TIA() {
			if read {
				return address & maskReadTIA, mem.TIA
			} else {
				return address & maskWriteTIA, mem.TIA
			}
		}
		return address, mem.MARIA
	}

	// it's not clear what addresses 0x0240 to 0x027f are mapped to

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

	// page 4
	if address >= 0x0300 && address <= 0x031f {
		// INPTCTRL or TIA
		address -= 0x0300
		if mem.INPTCTRL.Lock() {
			if read {
				return address & maskReadTIA, mem.TIA
			} else {
				return address & maskWriteTIA, mem.TIA
			}
		}
		return address, mem.INPTCTRL
	}
	if address >= 0x0320 && address <= 0x033f {
		// MARIA or TIA
		address -= 0x0300
		if mem.INPTCTRL.Lock() && mem.INPTCTRL.TIA() {
			if read {
				return address & maskReadTIA, mem.TIA
			} else {
				return address & maskWriteTIA, mem.TIA
			}
		}
		return address, mem.MARIA
	}

	// it's not clear what addresses 0x0340 to 0x037f are mapped to

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

	// 0x0400 to 0x047f "available for mapping by external devices"

	if address >= 0x0480 && address <= 0x04ff {
		// RAM RIOT
		return address - 0x0480, mem.RAMRIOT
	}

	// 0x0500 to 0x057f "available for mapping by external devices"

	if address >= 0x0580 && address <= 0x05ff {
		// RAM RIOT (shadow)
		return address - 0x0580, mem.RAMRIOT
	}

	// 0x0600 to 0x17ff "available for mapping by external devices"

	if address >= 0x1800 && address <= 0x27ff {
		// RAM 7800
		return address - 0x1800, mem.RAM7800
	}

	if mem.INPTCTRL.BIOS() {
		// BIOS
		if address >= bios.OriginBIOS && address <= 0xffff {
			return address - bios.OriginBIOS, mem.BIOS
		}
	}

	if address >= cartridge.OriginCart && address <= 0xffff {
		// cartridge
		return address - cartridge.OriginCart, mem.cartridge
	}

	return 0, nil
}

func (mem *Memory) Read(address uint16) (uint8, error) {
	idx, area := mem.MapAddress(address, true)
	if area == nil {
		return 0, fmt.Errorf("read unmapped address: %04x", address)
	}
	v, err := area.Read(idx)
	if err != nil {
		return 0, fmt.Errorf("read %04x: %w", address, err)
	}
	return v, nil
}

func (mem *Memory) Write(address uint16, data uint8) error {
	idx, area := mem.MapAddress(address, false)
	if area == nil {
		return fmt.Errorf("write unmapped address: %04x", address)
	}
	mem.Last = area
	err := area.Write(idx, data)
	if err != nil {
		return fmt.Errorf("write %04x: %w", address, err)
	}
	return nil
}
