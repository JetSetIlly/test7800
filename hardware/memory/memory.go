package memory

import (
	"fmt"

	"github.com/jetsetilly/test7800/hardware/memory/bios"
	"github.com/jetsetilly/test7800/hardware/memory/external"
	"github.com/jetsetilly/test7800/hardware/memory/inptctrl"
	"github.com/jetsetilly/test7800/hardware/memory/ram"
	"github.com/jetsetilly/test7800/hardware/riot"
	"github.com/jetsetilly/test7800/hardware/tia"
	"github.com/jetsetilly/test7800/logger"
)

type Memory struct {
	BIOS     *bios.BIOS
	INPTCTRL *inptctrl.INPTCTRL
	RAM7800  *ram.RAM
	RAMRIOT  *ram.RAM
	External *external.Device

	MARIA Area
	TIA   Area
	RIOT  Area

	// the last Area that was written to
	LastWrite Area

	// was last memory cycle an access of a TIA or RIOT address
	isTIA  bool
	isRIOT bool

	// most recent state of the address and data buses
	addressBus uint16
	dataBus    uint8
}

type Context interface {
	ram.Context
	external.Context
	Spec() string
}

// AddChips is returned by the Create() function and should be called to
// finalise the memory creation process
type AddChips func(maria Area, tia Area, riot Area)

type Area interface {
	Label() string

	// all valid memory areas must have an implementation of the Access()
	// function
	//
	// the write parameter represents the stats of the R/W line in the console.
	// individual memory areas can ignore this if it is not required or not
	// connected (2600 cartridge types)
	//
	// the address_or_idx parameter is either the raw unmapped address or an
	// index relative to the origin of the memory area. implementations of the
	// Area interface should make this clear by appropriate naming of the parameter.
	// more practically, the value of address_or_idx can be obtained through the
	// MapAddress() function
	//
	// the data parameter is the state of the data bus at the time of the call.
	// the write parameter indicates if the bus is being driven by the CPU at
	// the time of access (write will be true)
	//
	// the state of the data bus is returned along with an error. if the memory
	// area is not driving the data bus then the supplied data bus value should
	// be returned unchanged
	//
	// in rare cases where a memory area drives the data bus while the write
	// parameter is true (ie. the CPU is also driving the data bus) the conflict
	// should be resolved in the area implementation
	//
	// NOTE: the Access() function is provided instead of a separate Read/Write
	// functions. individual areas may have Read/Write functions for other
	// purposes (eg. the PEEK and POKE commands in a debugger) but these are not
	// necessary for emulation of memory
	//
	// for the more direct type of access the package level Read() and Write()
	// functions are provided
	Access(write bool, address_or_idx uint16, data uint8) (uint8, error)
}

func Create(ctx Context) (*Memory, AddChips) {
	var b bios.BIOS
	switch ctx.Spec() {
	case "PAL":
		b = bios.NewPAL()
	case "NTSC":
		b = bios.NewNTSC()
	default:
		b = bios.NewNTSC()
	}

	mem := &Memory{
		BIOS:     &b,
		INPTCTRL: &inptctrl.INPTCTRL{},
		RAM7800:  ram.Create(ctx, "ram7800", 0x1000),
		RAMRIOT:  ram.Create(ctx, "ramRIOT", 0x0080),
		External: external.Create(ctx),
	}
	return mem, func(maria Area, tia Area, riot Area) {
		mem.MARIA = maria
		mem.TIA = tia
		mem.RIOT = riot
	}
}

func (mem *Memory) IsSlow() bool {
	isSlow := mem.isTIA || mem.isRIOT
	mem.isTIA = false
	mem.isRIOT = false
	return isSlow
}

func (mem *Memory) Reset(random bool) {
	mem.INPTCTRL.Reset()
	mem.RAM7800.Reset(random)
	mem.RAMRIOT.Reset(random)
}

// MapAddress returns the memory "area" and a "mapped" address for the area
// corresponding to the address. the "mapped" address can either be a normalised
// address relative to $0000 or an index relative to the origin of the area.
//
// The result partially depends on the state of INPTCTRL. It will always return
// INPTCTRL, MARIA, etc. unless the Lock() is true and TIA() is true, in which
// case it may return TIA, INPTCTRL or MARIA depending on the address.
//
// It is possible for a nil Area to be returned. In which case, the index value
// will be zero.
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
			return address, mem.TIA
		}
		return address, mem.INPTCTRL
	}
	if address >= 0x0020 && address <= 0x003f {
		// MARIA or TIA
		if mem.INPTCTRL.Lock() && mem.INPTCTRL.TIA() {
			return address, mem.TIA
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
			return address, mem.TIA
		}
		return address, mem.INPTCTRL
	}
	if address >= 0x0120 && address <= 0x013f {
		// MARIA or TIA
		address -= 0x0100
		if mem.INPTCTRL.Lock() && mem.INPTCTRL.TIA() {
			return address, mem.TIA
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
			return address, mem.TIA
		}
		return address, mem.INPTCTRL
	}
	if address >= 0x0220 && address <= 0x023f {
		// MARIA or TIA
		address -= 0x0200
		if mem.INPTCTRL.Lock() && mem.INPTCTRL.TIA() {
			return address, mem.TIA
		}
		return address, mem.MARIA
	}

	// it's not clear what addresses 0x0240 to 0x027f are mapped to but for now,
	// I'll assume it's a shadow of page 0 which means it's RAM 7800 block 0 but
	// only part of it
	if address >= 0x0240 && address <= 0x027f {
		// RAM 7800 block 0 (partial)
		return address - 0x0240 + 0x0840, mem.RAM7800
	}

	if address >= 0x0280 && address <= 0x02ff {
		// RIOT
		return address - 0x0280, mem.RIOT
	}

	// page 4
	if address >= 0x0300 && address <= 0x031f {
		// INPTCTRL or TIA
		address -= 0x0300
		if mem.INPTCTRL.Lock() {
			return address, mem.TIA
		}
		return address, mem.INPTCTRL
	}
	if address >= 0x0320 && address <= 0x033f {
		// MARIA or TIA
		address -= 0x0300
		if mem.INPTCTRL.Lock() && mem.INPTCTRL.TIA() {
			return address, mem.TIA
		}
		return address, mem.MARIA
	}

	// it's not clear what addresses 0x0340 to 0x037f are mapped to but for now,
	// I'll assume it's a shadow of page 1 which means it's RAM 7800 block 1 but
	// only part of it
	if address >= 0x0340 && address <= 0x037f {
		// RAM 7800 block 1 (partial)
		return address - 0x0340 + 0x0940, mem.RAM7800
	}

	if address >= 0x0380 && address <= 0x03ff {
		// RIOT
		return address - 0x0380, mem.RIOT
	}

	// 0x0400 to 0x047f "available for mapping by external devices"
	if address >= 0x0400 && address <= 0x047f {
		// external
		return address, mem.External
	}

	if address >= 0x0480 && address <= 0x04ff {
		// RAM RIOT
		return address - 0x0480, mem.RAMRIOT
	}

	// 0x0500 to 0x057f "available for mapping by external devices"
	if address >= 0x0500 && address <= 0x057f {
		// external
		return address, mem.External
	}

	if address >= 0x0580 && address <= 0x05ff {
		// RAM RIOT (shadow)
		return address - 0x0580, mem.RAMRIOT
	}

	// 0x0600 to 0x17ff "available for mapping by external devices"
	if address >= 0x0600 && address <= 0x17ff {
		// external
		return address, mem.External
	}

	if address >= 0x1800 && address <= 0x27ff {
		// RAM 7800
		return address - 0x1800, mem.RAM7800
	}

	if address >= 0x2800 && address <= 0x2fff {
		// 0x2800 to 0x2fff is considered to be an unreliable mirror of RAM 7800

		// there's a very specific reason to believe the range 0x2800 to 0x28ff
		// is a valid mirror
		// https://forums.atariage.com/topic/370030-does-anyone-abuse-the-ram-mirrors/#findComment-5507270
		if address >= 0x2800 && address <= 0x28ff {
			return address - 0x1900, mem.RAM7800
		}

		return 0, nil
	}

	if address >= 0x3000 && address <= 0x7fff {
		return address, mem.External
	}

	if address >= 0x8000 && address <= 0xffff {
		if mem.INPTCTRL.BIOS() {
			return address, mem.BIOS
		}
	}

	// everything else can be handled by the external package
	return address, mem.External
}

// Read memory address as viewed by the CPU. This method of reading creates
// side-effects to the memory system. For reading of memory without side-effects
// use the package level Read() function
func (mem *Memory) Read(address uint16) (uint8, error) {
	if mem.addressBus != address {
		mem.addressBus = address
		err := mem.External.BusChange(mem.addressBus, mem.dataBus)
		if err != nil {
			return 0, fmt.Errorf("read: %w", err)
		}
	}

	idx, area := mem.MapAddress(address, true)
	if area == nil {
		logger.Logf(logger.Allow, "memory read", "unmapped address: %02x", address)
		return 0, nil
	}

	_, mem.isTIA = area.(*tia.TIA)
	_, mem.isRIOT = area.(*riot.RIOT)

	data, err := area.Access(false, idx, mem.dataBus)
	if err != nil {
		return 0, fmt.Errorf("read %04x: %w", address, err)
	}

	// data bus happens late for Read()
	mem.dataBus = data

	return data, nil
}

// Write memory address as driven by the CPU. This method of writing produces
// side-effects to the memory system, in addition to the explicit writing of
// data. For writing to memory without side-effects use the package level
// Write() function
func (mem *Memory) Write(address uint16, data uint8) error {
	// data bus happens early for Write()
	mem.dataBus = data

	if mem.addressBus != address {
		mem.addressBus = address
		err := mem.External.BusChange(mem.addressBus, mem.dataBus)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
	}

	idx, area := mem.MapAddress(address, false)
	if area == nil {
		logger.Logf(logger.Allow, "memory write", "unmapped address: %02x", address)
		return nil
	}

	_, mem.isTIA = area.(*tia.TIA)
	mem.LastWrite = area

	data, err := area.Access(true, idx, data)
	if err != nil {
		return fmt.Errorf("write %04x: %w", address, err)
	}

	return nil
}

// Read memory of the specified area. This method of reading memory is for more
// direct uses (eg. a debugger PEEK command) and does not effect the general
// state of memory
//
// The area and address parameters should be acquired from the MapAddress()
// function
func Read(area Area, address uint16) (uint8, error) {
	return area.Access(false, address, 0)
}

// Write a value to the specified address of the area. This method of writing
// memory is for more direct uses (eg. a debugger POKE command) and does not
// effect the general state of memory
//
// The area and address parameters should be acquired from the MapAddress()
// function
func Write(area Area, address uint16, data uint8) error {
	_, err := area.Access(true, address, data)
	return err
}
