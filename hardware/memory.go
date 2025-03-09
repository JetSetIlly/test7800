package hardware

import (
	"fmt"

	"github.com/jetsetilly/test7800/hardware/bios"
	"github.com/jetsetilly/test7800/hardware/inptctrl"
	"github.com/jetsetilly/test7800/hardware/maria"
	"github.com/jetsetilly/test7800/hardware/ram"
)

type lastArea interface {
	Label() string
	Status() string
}

type memory struct {
	bios     bios.BIOS
	INPTCTRL inptctrl.INPTCTRL
	MARIA    maria.Maria
	RAM7800  ram.RAM
	RAMRIOT  ram.RAM
	last     lastArea
}

type Area interface {
	// read and write both take an index value. this is an address in the area
	// but with the area origin removed. in other words, the area doesn't need
	// to know about it's location in memory, only the relative placement of
	// addresses within the area
	Read(idx uint16) (uint8, error)
	Write(idx uint16, data uint8) error
}

func (mem *memory) MapAddress(address uint16) (uint16, Area) {
	// page one
	if address >= 0x0000 && address <= 0x001f {
		// INPTCTRL
		return address, &mem.INPTCTRL
	}
	if address >= 0x0020 && address <= 0x003f {
		// MARIA
		return address, &mem.MARIA
	}
	if address >= 0x0040 && address <= 0x00ff {
		// RAM 7800 block 0
		return address - 0x0040 + 0x0840, &mem.RAM7800
	}

	// page 2
	if address >= 0x0100 && address <= 0x011f {
		// INPTCTRL
		return address - 0x0100, &mem.INPTCTRL
	}
	if address >= 0x0120 && address <= 0x013f {
		// MARIA
		return address - 0x0120, &mem.MARIA
	}
	if address >= 0x0140 && address <= 0x01ff {
		// RAM 7800 block 1
		return address - 0x0140 + 0x0940, &mem.RAM7800
	}

	// unsure

	if address >= 0x0280 && address <= 0x02ff {
		// RIOT IO
	}

	// unsure

	if address >= 0x0380 && address <= 0x03ff {
		// RIOT IO
	}

	// unsure

	if address >= 0x0480 && address <= 0x04ff {
		// RAM RIOT
		return address - 0x0480, &mem.RAMRIOT
	}

	// unsure

	if address >= 0x1800 && address <= 0x27ff {
		// RAM 7800
		return address - 0x1800, &mem.RAM7800
	}

	if address >= 0xf000 && address <= 0xffff {
		// BIOS
		return address & 0xfff, &mem.bios
	}

	return 0, nil
}

func (mem *memory) Read(address uint16) (uint8, error) {
	idx, area := mem.MapAddress(address)
	if area == nil {
		return 0, fmt.Errorf("memory.Read: unmapped address: %04x", address)
	}
	return area.Read(idx)
}

func (mem *memory) Write(address uint16, data uint8) error {
	idx, area := mem.MapAddress(address)
	if area == nil {
		return fmt.Errorf("memory.Write: unmapped address: %04x", address)
	}
	if l, ok := area.(lastArea); ok {
		mem.last = l
	}
	return area.Write(idx, data)
}
