package hardware

import (
	"fmt"

	"github.com/jetsetilly/test7800/hardware/bios"
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
	bios     *bios.BIOS
	INPTCTRL *inptctrl.INPTCTRL
	MARIA    *maria.Maria
	RAM7800  *ram.RAM
	RAMRIOT  *ram.RAM
	TIA      *tia.TIA
	RIOT     *riot.RIOT
	last     lastArea
}

func createMemory() *memory {
	mem := &memory{
		bios:     &bios.BIOS{},
		INPTCTRL: &inptctrl.INPTCTRL{},
		MARIA:    &maria.Maria{},
		RAM7800:  ram.Create("ram7800", 0x1000),
		RAMRIOT:  ram.Create("ramRIOT", 0x0080),
		TIA:      &tia.TIA{},
		RIOT:     &riot.RIOT{},
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
}

// MapAddress partially depends on the state of INPTCTRL. It will always return
// INPTCTRL, MARIA, etc. unless the Lock() is true and MARIA() is false, in
// which case it may return TIA or nil depending on the address
//
// There is also a flaw in this emulation but the flaw only affects the BIOS.
// The flaw is in this section of the BIOS:
//
// (In this code STARTVND corresponds to address $fb17)
//
//	STARTVND  LDX     #STACKPTR
//	          TXS                            ;SET STACK POINTER
//
//	          LDA     #0                     ;ZERO THE TIA REGISTERS OUT
//	          TAX
//	TIA0LOOP  STA     1,X
//	          INX
//	          CPX     #$2C
//	          BNE     TIA0LOOP
//	          LDA     #$02                   ;BACK INTO MARIA MODE
//	          STA     INPTCTRL
//
// For this code to work as intended (the TIA registers zeroed) then the data
// must be forwarded to the TIA in addition to the INPTCTRL. If not, then the
// TIA registers would not be cleared.
//
// Once the INPTCTRL lock is engaged then the mapping is straight-forward. The
// lock will always be engaged during the BIOS.
func (mem *memory) MapAddress(address uint16) (uint16, Area) {
	// page one
	if address >= 0x0000 && address <= 0x001f {
		// INPTCTRL or TIA
		if mem.INPTCTRL.Lock() && !mem.INPTCTRL.MARIA() {
			return address, mem.TIA
		}
		return address, mem.INPTCTRL
	}
	if address >= 0x0020 && address <= 0x003f {
		// MARIA or TIA
		if mem.INPTCTRL.Lock() && !mem.INPTCTRL.MARIA() {
			if address <= 0x002f {
				return address, mem.TIA
			} else {
				return 0, nil
			}
		}
		return address, mem.MARIA
	}
	if address >= 0x0040 && address <= 0x00ff {
		// RAM 7800 block 0
		if mem.INPTCTRL.Lock() && !mem.INPTCTRL.MARIA() {
			return 0, nil
		}
		return address - 0x0040 + 0x0840, mem.RAM7800
	}

	// page 2
	if address >= 0x0100 && address <= 0x011f {
		if mem.INPTCTRL.Lock() && !mem.INPTCTRL.MARIA() {
			return address - 0x0100, mem.TIA
		}
		return address - 0x0100, mem.INPTCTRL
	}
	if address >= 0x0120 && address <= 0x013f {
		// MARIA or TIA
		if mem.INPTCTRL.Lock() && !mem.INPTCTRL.MARIA() {
			if address <= 0x012f {
				return address - 0x120, mem.TIA
			} else {
				return 0, nil
			}
		}
		return address - 0x120, mem.MARIA
	}
	if address >= 0x0140 && address <= 0x01ff {
		// RAM 7800 block 1
		if mem.INPTCTRL.Lock() && !mem.INPTCTRL.MARIA() {
			return 0, nil
		}
		return address - 0x0140 + 0x0940, mem.RAM7800
	}

	// unsure

	if address >= 0x0280 && address <= 0x02ff {
		// RIOT
		return address - 0x0280, mem.RIOT
	}

	// unsure

	if address >= 0x0380 && address <= 0x03ff {
		// RIOT
		return address - 0x0380, mem.RIOT
	}

	// unsure

	if address >= 0x0480 && address <= 0x04ff {
		// RAM RIOT
		return address - 0x0480, mem.RAMRIOT
	}

	// unsure

	if address >= 0x1800 && address <= 0x27ff {
		// RAM 7800
		if mem.INPTCTRL.Lock() && !mem.INPTCTRL.MARIA() {
			return 0, nil
		}
		return address - 0x1800, mem.RAM7800
	}

	if address >= 0xf000 && address <= 0xffff {
		// BIOS
		return address - 0xf000, mem.bios
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
