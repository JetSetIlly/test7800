package main

import (
	"fmt"
	"math/rand/v2"

	_ "embed"

	"github.com/jetsetilly/test7800/cpu"
	"github.com/jetsetilly/test7800/maria"
)

//go:embed "7800 BIOS (U).rom"
var biosrom []byte

type bios struct {
}

func (b *bios) Label() string {
	return "BIOS"
}

func (b *bios) Read(idx uint16) (uint8, error) {
	return biosrom[idx], nil
}

func (b *bios) Write(_ uint16, data uint8) error {
	return nil
}

type inptctrl struct {
	value uint8
}

func (ic *inptctrl) Label() string {
	return "INPTCTRL"
}

func (ic *inptctrl) Status() string {
	lock := ic.value&0x01 == 0x01
	maria := ic.value&0x02 == 0x02
	bios := ic.value&0x04 == 0x04
	tia := ic.value&0x08 == 0x08
	return fmt.Sprintf("INPTCTRL: lock=%v maria=%v bios=%v tia=%v", lock, maria, bios, tia)
}

func (ic *inptctrl) Read(_ uint16) (uint8, error) {
	return ic.value, nil
}

func (ic *inptctrl) Write(_ uint16, data uint8) error {
	if ic.value&0x01 == 0x01 {
		return nil
	}
	ic.value = data
	return nil
}

type ram struct {
	label string
	data  []uint8
}

func (r *ram) Label() string {
	return r.label
}

func (r *ram) Read(idx uint16) (uint8, error) {
	return r.data[idx], nil
}

func (r *ram) Write(idx uint16, data uint8) error {
	r.data[idx] = data
	return nil
}

type lastArea interface {
	Label() string
	Status() string
}

type memory struct {
	bios     bios
	inptctrl inptctrl
	maria    maria.Maria
	ram7800  ram
	ramRIOT  ram
	last     lastArea
}

type area interface {
	// read and write both take an index value. this is an address in the area
	// but with the area origin removed. in other words, the area doesn't need
	// to know about it's location in memory, only the relative placement of
	// addresses within the area
	Read(idx uint16) (uint8, error)
	Write(idx uint16, data uint8) error
}

func (mem *memory) mapAddress(address uint16) (uint16, area) {
	// page one
	if address >= 0x0000 && address <= 0x001f {
		// INPTCTRL
		return address, &mem.inptctrl
	}
	if address >= 0x0020 && address <= 0x003f {
		// MARIA
		return address, &mem.maria
	}
	if address >= 0x0040 && address <= 0x00ff {
		// RAM 7800 block 0
		return address - 0x0040 + 0x0840, &mem.ram7800
	}

	// page 2
	if address >= 0x0100 && address <= 0x011f {
		// INPTCTRL
		return address - 0x0100, &mem.inptctrl
	}
	if address >= 0x0120 && address <= 0x013f {
		// MARIA
		return address - 0x0120, &mem.maria
	}
	if address >= 0x0140 && address <= 0x01ff {
		// RAM 7800 block 1
		return address - 0x0140 + 0x0940, &mem.ram7800
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
		return address - 0x0480, &mem.ramRIOT
	}

	// unsure

	if address >= 0x1800 && address <= 0x27ff {
		// RAM 7800
		return address - 0x1800, &mem.ram7800
	}

	if address >= 0xf000 && address <= 0xffff {
		// BIOS
		return address & 0xfff, &mem.bios
	}

	return 0, nil
}

func (mem *memory) Read(address uint16) (uint8, error) {
	idx, area := mem.mapAddress(address)
	if area == nil {
		return 0, fmt.Errorf("memory.Read: unmapped address: %04x", address)
	}
	return area.Read(idx)
}

func (mem *memory) Write(address uint16, data uint8) error {
	idx, area := mem.mapAddress(address)
	if area == nil {
		return fmt.Errorf("memory.Write: unmapped address: %04x", address)
	}
	if l, ok := area.(lastArea); ok {
		mem.last = l
	}
	return area.Write(idx, data)
}

type console struct {
	mc  *cpu.CPU
	mem *memory
}

func (con *console) initialise() {
	con.mem = &memory{
		ram7800: ram{
			label: "ram7800",
			data:  make([]uint8, 0x1000),
		},
		ramRIOT: ram{
			label: "ramRIOT",
			data:  make([]uint8, 0x0080),
		},
	}
	con.mc = cpu.NewCPU(con.mem)
	con.reset(true)
}

func (con *console) reset(random bool) error {
	con.mc.Reset()

	if random {
		for i := range len(con.mem.ram7800.data) {
			con.mem.ram7800.data[i] = uint8(rand.IntN(255))
		}
		for i := range len(con.mem.ramRIOT.data) {
			con.mem.ramRIOT.data[i] = uint8(rand.IntN(255))
		}
		con.mc.PC.Load(uint16(rand.IntN(65535)))
		con.mc.A.Load(uint8(rand.IntN(255)))
		con.mc.X.Load(uint8(rand.IntN(255)))
		con.mc.Y.Load(uint8(rand.IntN(255)))
	} else {
		clear(con.mem.ram7800.data)
		clear(con.mem.ramRIOT.data)
	}

	return con.mc.LoadPCIndirect(cpu.Reset)
}

func (con *console) step() error {
	cycle := func() error {
		return nil
	}
	return con.mc.ExecuteInstruction(cycle)
}

func (con *console) lastMemoryAccess() string {
	if con.mem.last == nil {
		return ""
	}
	s := con.mem.last.Status()
	con.mem.last = nil
	return s
}
