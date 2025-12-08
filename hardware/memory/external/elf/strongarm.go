// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package elf

import (
	"github.com/jetsetilly/test7800/hardware/arm"
	"github.com/jetsetilly/test7800/logger"
)

// signature of a strongarm function. a pointer to an instance of elfMemory is
// passed as an argument, rather than the function being a memory of elfMemory.
// this makes the Plumb() function far simpler.
type strongArmFunction func(*elfMemory)

// the strongarm function specification lists the implementation function and
// any meta-information for a single strongarm function
type strongArmFunctionSpec struct {
	name     string
	function strongArmFunction
	support  bool
}

// strongarm function state records the progress of a single strongarm function
type strongArmFunctionState struct {
	function  strongArmFunction
	state     int
	registers [arm.NumCoreRegisters]uint32

	// the vcsCopyOverblankToRiotRam() function is a loop. we need to keep
	// track of the loop counter and sub-state in addition to the normal state
	// value
	//
	// the mechanism can be used for other looping functions
	counter    int
	subCounter int
}

// state of the strongarm emulation. not all ELF binaries make uses of the
// strongarm functions, in those instances strongArmState will be unused
type strongArmState struct {
	running strongArmFunctionState

	// the expected next 6507 address to be working with
	nextRomAddress uint16

	// bus stuffing
	lowMask          uint8
	correctionMaskHi uint8
	correctionMaskLo uint8

	opcodeLookup [256]uint8
	modeLookup   [256]uint8
}

// strongARM functions need to return to the main program with a branch exchange
var strongArmStub = []byte{
	0x70, 0x47, // BX LR
	0x00, 0x00,
}

func (mem *elfMemory) setNextRomAddress(addr uint16) {
	mem.strongarm.nextRomAddress = addr & Memtop
}

func (mem *elfMemory) injectRomByte(data uint8) bool {
	if mem.stream.active {
		mem.stream.push(streamEntry{
			addr: mem.strongarm.nextRomAddress,
			data: data,
		})
		mem.strongarm.nextRomAddress++
		return true
	}

	addrIn := uint16(mem.gpio.data[ADDR_IDR])
	addrIn |= uint16(mem.gpio.data[ADDR_IDR+1]) << 8
	addrIn &= Memtop

	if addrIn != mem.strongarm.nextRomAddress {
		return false
	}

	mem.gpio.data[DATA_ODR] = data
	mem.strongarm.nextRomAddress++

	return true
}

// injectBusStuff adds bus stuff data into the stream
func (mem *elfMemory) injectBusStuff(data uint8) {
	if mem.stream.active {
		mem.stream.push(streamEntry{
			data:     data,
			busstuff: true,
		})
		return
	}
	mem.busStuff = true
	mem.busStuffData = data
}

func (mem *elfMemory) yieldDataBus(addr uint16) bool {
	if mem.stream.active {
		return true
	}

	addrIn := uint16(mem.gpio.data[ADDR_IDR])
	addrIn |= uint16(mem.gpio.data[ADDR_IDR+1]) << 8
	addrIn &= Memtop
	addr &= Memtop

	if addrIn != addr {
		return false
	}

	return true
}

func (mem *elfMemory) yieldDataBusToStack() bool {
	if mem.stream.active {
		return true
	}

	addrIn := uint16(mem.gpio.data[ADDR_IDR])
	addrIn |= uint16(mem.gpio.data[ADDR_IDR+1]) << 8
	addrIn &= Memtop

	if addrIn&0xfe00 != 0 {
		return false
	}

	return true
}

func snoopDataBus(mem *elfMemory) {
	addrIn := uint16(mem.gpio.data[ADDR_IDR])
	addrIn |= uint16(mem.gpio.data[ADDR_IDR+1]) << 8
	addrIn &= Memtop

	if addrIn == mem.strongarm.nextRomAddress {
		// setting return value
		mem.arm.RegisterSet(0, uint32(mem.gpio.data[DATA_IDR]))

		// continue with additional NOP or end strongarm immediately
		if mem.followSnoopBusWithNOP {
			mem.followSnoopBusWithNOP = false
			mem.runStrongArmFunction(vcsNop2)
		} else {
			mem.endStrongArmFunction()
		}
	}

	// note that this implementation of snoopDataBus is missing the "give
	// peripheral time to respond" loop that we see in the real vcsLib
}

// snoopDataBus is significantly different when streaming is enabled
func snoopDataBus_streaming(mem *elfMemory, addr uint16) {
	if addr == mem.stream.snoopDataBusAddr {
		mem.arm.RegisterSet(0, uint32(mem.gpio.data[DATA_IDR]))
		mem.stream.snoopDataBus = nil

		// continue with additional NOP or end strongarm immediately
		if mem.followSnoopBusWithNOP {
			mem.followSnoopBusWithNOP = false
			mem.runStrongArmFunction(vcsNop2)
		}
	}
}

func (str *strongArmState) updateLookupTables() {
	for i := 0; i < 256; i++ {
		if uint8(i)&str.correctionMaskHi == str.correctionMaskHi {
			if uint8(i)&str.correctionMaskLo == str.correctionMaskLo {
				str.opcodeLookup[i] = 0x84
			} else {
				str.opcodeLookup[i] = 0x86
			}
		} else {
			if uint8(i)&str.correctionMaskLo == str.correctionMaskLo {
				str.opcodeLookup[i] = 0x85
			} else {
				str.opcodeLookup[i] = 0x87
			}
		}

		mode := uint8(i) ^ str.lowMask

		// never drive the bits that get corrected by opcodes above
		mode &= ^str.correctionMaskLo
		mode &= ^str.correctionMaskHi

		str.modeLookup[i] = ((mode & 0x80) << 7) |
			((mode & 0x40) << 6) |
			((mode & 0x20) << 5) |
			((mode & 0x10) << 4) |
			((mode & 0x08) << 3) |
			((mode & 0x04) << 2) |
			((mode & 0x02) << 1) |
			(mode & 0x01)
	}
}

// initialise state ready for bus stuffing. we know bus stuffing is used if the
// vcsWrite3() function has been detected (during relocation).
func (mem *elfMemory) busStuffingInit() {
	if !mem.usesBusStuffing {
		logger.Log(logger.Allow, "ELF", "ROM does not use any bus stuffing instructions")
		return
	}

	logger.Log(logger.Allow, "ELF", "ROM uses bus stuffing instructions")

	mem.strongarm.lowMask = 0xff
	mem.strongarm.correctionMaskHi = 0x00
	mem.strongarm.correctionMaskLo = 0x00
	mem.strongarm.updateLookupTables()
}

// setStrongArmFunction initialises the next function to run. it takes a copy
// of the ARM registers at that point of initialisation. the register values
// are used to supply arguments to the strongArmFunction, as many as the
// function requires (up to 32). any arguments provided to the function will
// be used instead of the corresponding register value (numbered from 0 to 31)
func (mem *elfMemory) setStrongArmFunction(f strongArmFunction, args ...uint32) {
	mem.strongarm.running.function = f
	mem.strongarm.running.state = 0
	mem.strongarm.running.registers = mem.arm.CoreRegisters()
	copy(mem.strongarm.running.registers[:], args)
}

// runStrongArmFunction initialises the next function to run and immediatly
// executes it
//
// it differs to setStrongArmFunction in that the function does not cause the
// ARM to yield to the VCS
func (mem *elfMemory) runStrongArmFunction(f strongArmFunction, args ...uint32) {
	mem.strongarm.running.registers = mem.arm.CoreRegisters()
	copy(mem.strongarm.running.registers[:], args)
	f(mem)
}

// a strongArmFunction should always end with a call to endFunction() no matter
// how many execution states it has.
func (mem *elfMemory) endStrongArmFunction() {
	mem.strongarm.running.function = nil
}
