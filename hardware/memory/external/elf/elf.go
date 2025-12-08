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
	"crypto/md5"
	"debug/elf"
	"encoding/binary"
	"fmt"

	"github.com/jetsetilly/test7800/coprocessor"
	"github.com/jetsetilly/test7800/hardware/arm"
	"github.com/jetsetilly/test7800/hardware/cpu"
	"github.com/jetsetilly/test7800/hardware/spec"
)

const (
	CartridgeBits = 0x0fff
	OriginCart    = 0xf000
	Memtop        = 0x1fff

	// preferences
	random = true
)

type Context interface {
	Rand8Bit() uint8
	Break(e error)
	Spec() spec.Spec
	IsAtari7800() bool
}

type Elf struct {
	ctx     Context
	version string

	arm *arm.ARM
	mem *elfMemory

	// the hook that handles cartridge yields
	yieldHook coprocessor.CartYieldHook
}

// elfReaderAt is an implementation of io.ReaderAt and is used with elf.NewFile()
type elfReaderAt struct {
	// data from the file being used as the source of ELF data
	data []byte

	// the offset into the data slice where the ELF file starts
	offset int64
}

func (r *elfReaderAt) ReadAt(p []byte, start int64) (n int, err error) {
	start += r.offset

	end := start + int64(len(p))
	end = min(int64(len(r.data)), end)
	copy(p, r.data[start:end])

	n = int(end - start)
	if n < len(p) {
		return n, fmt.Errorf("not enough bytes in the ELF data to fill the buffer")
	}

	return n, nil
}

// NewElf is the preferred method of initialisation for the Elf type.
func NewElf(ctx Context, d []byte) (*Elf, error) {
	r := &elfReaderAt{data: d}

	// ELF file is read via our elfReaderAt instance
	ef, err := elf.NewFile(r)
	if err != nil {
		return nil, fmt.Errorf("ELF: %w", err)
	}
	defer ef.Close()

	// keeping things simple. only 32bit ELF files supported
	if ef.Class != elf.ELFCLASS32 {
		return nil, fmt.Errorf("ELF: only 32bit ELF files are supported")
	}

	// sanity checks on ELF data
	if ef.FileHeader.Machine != elf.EM_ARM {
		return nil, fmt.Errorf("ELF: is not ARM")
	}
	if ef.FileHeader.Version != elf.EV_CURRENT {
		return nil, fmt.Errorf("ELF: unknown version")
	}
	if ef.FileHeader.Type != elf.ET_REL {
		return nil, fmt.Errorf("ELF: is not relocatable")
	}

	// big endian byte order is probably fine but we've not tested it
	if ef.FileHeader.ByteOrder != binary.LittleEndian {
		return nil, fmt.Errorf("ELF: is not little-endian")
	}

	cart := &Elf{
		ctx:       ctx,
		yieldHook: coprocessor.StubCartYieldHook{},
	}

	cart.mem = newElfMemory(ctx)
	cart.arm = arm.NewARM(cart.mem.model, cart.mem, cart)
	cart.arm.CycleDuringImmediateMode(true)
	cart.mem.arm = cart.arm
	err = cart.mem.decode(ef)
	if err != nil {
		return nil, err
	}
	cart.mem.md5sum = fmt.Sprintf("%x", md5.Sum(d))

	cart.arm.SetByteOrder(ef.ByteOrder)
	cart.mem.busStuffingInit()

	// defer VCS reset until the VCS tries to read the reset address

	// run arm initialisation functions if present. next call to arm.Run() will
	// cause the main function to execute
	err = cart.mem.runInitialisation(cart.arm)
	if err != nil {
		return nil, fmt.Errorf("ELF: %w", err)
	}

	return cart, nil
}

func (cart *Elf) Label() string {
	return "ELF"
}

// reset is distinct from Reset(). this reset function is implied by the
// reading of the reset address.
func (cart *Elf) reset() {
	// stream bytes rather than injecting them into the VCS as they arrive
	cart.mem.stream.active = !cart.mem.stream.disabled

	// initialise ROM for the VCS
	if cart.mem.stream.active {
		cart.mem.stream.push(streamEntry{
			addr: 0x1ffc,
			data: 0x00,
		})
		cart.mem.stream.push(streamEntry{
			addr: 0x1ffd,
			data: 0x10,
		})
		cart.mem.strongarm.nextRomAddress = 0x1000
		cart.mem.stream.startDrain()
	} else {
		cart.mem.setStrongArmFunction(vcsLibInit)
	}

	// set arguments for initial execution of ARM program
	systemType := argSystemType_NTSC_7800
	switch cart.ctx.Spec().ID {
	case "NTSC":
		if cart.ctx.IsAtari7800() {
			systemType = argSystemType_NTSC_7800
		} else {
			systemType = argSystemType_NTSC
		}
	case "PAL":
		if cart.ctx.IsAtari7800() {
			systemType = argSystemType_PAL_7800
		} else {
			systemType = argSystemType_PAL
		}
	case "PAL60":
		systemType = argSystemType_PAL60
	}

	flags := argFlags_NoExit

	binary.LittleEndian.PutUint32(cart.mem.args[argAddrSystemType-argOrigin:], uint32(systemType))
	binary.LittleEndian.PutUint32(cart.mem.args[argAddrClockHz-argOrigin:], uint32(cart.arm.Clk))
	binary.LittleEndian.PutUint32(cart.mem.args[argAddrFlags-argOrigin:], uint32(flags))

	cart.arm.SetInitialRegisters(argOrigin)
}

func (cart *Elf) Access(_ bool, addr uint16, _ uint8) (uint8, error) {
	if cart.mem.stream.active {
		if !cart.mem.stream.drain {
			if cart.runARM(addr) {
				return 0, nil
			}
		}

		if addr&CartridgeBits == cart.mem.stream.peek().addr&CartridgeBits {
			e := cart.mem.stream.pull()
			cart.mem.gpio.data[DATA_ODR] = e.data
		}
	}

	cart.mem.busStuffDelay = true
	return cart.mem.gpio.data[DATA_ODR], nil
}

func (cart *Elf) NumBanks() int {
	return 1
}

func (cart *Elf) runARM(addr uint16) bool {
	if cart.mem.stream.active {
		// do nothing with the ARM if the byte stream is draining
		if cart.mem.stream.drain {
			return true
		}

		// run preempted snoopDataBus() function if required
		if cart.mem.stream.snoopDataBus != nil {
			cart.mem.stream.snoopDataBus(cart.mem, addr)
			return true
		}
	}

	cart.arm.StartProfiling()
	defer cart.arm.ProcessProfiling()

	// call arm once and then check for yield conditions
	cart.mem.yield, _ = cart.arm.Run()

	// keep calling runArm() for as long as program does not need to sync with the VCS
	for cart.mem.yield.Type != coprocessor.YieldSyncWithVCS {
		// the ARM should never return YieldProgramEnded when executing code
		// from the ELF type. if it does then it is an error and we should yield
		// with YieldExecutionError
		if cart.mem.yield.Type == coprocessor.YieldProgramEnded {
			cart.mem.yield.Type = coprocessor.YieldExecutionError
			cart.mem.yield.Error = fmt.Errorf("ELF does not support program-ended yield")
		}

		// treat infinite loops like a YieldSyncWithVCS
		if cart.mem.yield.Type == coprocessor.YieldInfiniteLoop {
			return true
		}

		switch cart.yieldHook.CartYield(cart.mem.yield) {
		case coprocessor.YieldHookEnd:
			return false
		case coprocessor.YieldHookContinue:
			cart.mem.yield, _ = cart.arm.Run()
		}
	}

	return true
}

func (cart *Elf) BusChange(addr uint16, data uint8) error {
	if cart.mem.stream.active && cart.mem.stream.drain {
		return nil
	}

	// if memory access is not a cartridge address (ie. a TIA or RIOT address)
	// then the ARM is running in parallel (ie. no synchronisation)
	cart.mem.parallelARM = (addr&OriginCart != OriginCart)

	// reset address with any mirror origin
	const resetAddrAnyMirror = (cpu.Reset & CartridgeBits) | OriginCart

	// if address is the reset address then trigger the reset procedure
	if (addr&CartridgeBits)|OriginCart == resetAddrAnyMirror {
		// after this call to cart reset, the cartridge will be wanting to run
		// the vcsEmulationInit() strongarm function
		cart.reset()
	}

	// set GPIO data and address information
	cart.mem.gpio.data[DATA_IDR] = data
	cart.mem.gpio.data[ADDR_IDR] = uint8(addr)
	cart.mem.gpio.data[ADDR_IDR+1] = uint8(addr >> 8)

	// if byte-streaming is active then the access is relatively simple
	if cart.mem.stream.active {
		_ = cart.runARM(addr)
		return nil
	}

	// handle ARM synchronisation for non-byte-streaming mode. the sequence of
	// calls to runARM() and whatever strongarm function might be active was
	// arrived through experimentation. a more efficient way of doing this
	// hasn't been discovered yet

	runStrongarm := func() bool {
		if cart.mem.strongarm.running.function == nil {
			return false
		}
		cart.mem.strongarm.running.function(cart.mem)
		if cart.mem.strongarm.running.function == nil {
			if !cart.runARM(addr) {
				return false
			}
			if cart.mem.strongarm.running.function != nil {
				cart.mem.strongarm.running.function(cart.mem)
			}
		}
		return true
	}

	if runStrongarm() {
		return nil
	}

	if !cart.runARM(addr) {
		return nil
	}
	if runStrongarm() {
		return nil
	}

	if !cart.runARM(addr) {
		return nil
	}
	if runStrongarm() {
		return nil
	}

	if !cart.runARM(addr) {
		return nil
	}

	return nil
}

func (cart *Elf) Step(clock float32) {
	cart.arm.Step(clock)
}

func (cart *Elf) ARMinterrupt(addr uint32, val1 uint32, val2 uint32) (arm.ARMinterruptReturn, error) {
	return arm.ARMinterruptReturn{}, nil
}

func (cart *Elf) BusStuff() (uint8, bool) {
	if !cart.mem.usesBusStuffing {
		return 0, false
	}

	if cart.mem.busStuffDelay {
		cart.mem.busStuffDelay = false
		return 0, false
	}

	if cart.mem.stream.active {
		if cart.mem.stream.peek().busstuff {
			e := cart.mem.stream.pull()
			return e.data, true
		}
		return 0, false
	}

	if cart.mem.busStuff {
		cart.mem.busStuff = false
		return cart.mem.busStuffData, true
	}
	return 0, false
}

func (cart *Elf) Section(name string) ([]uint8, uint32) {
	if idx, ok := cart.mem.sectionsByName[name]; ok {
		s := cart.mem.sections[idx]
		return s.data, s.origin
	}
	return nil, 0
}

func (cart *Elf) CoProcExecutionState() coprocessor.CoProcExecutionState {
	if cart.mem.parallelARM {
		return coprocessor.CoProcExecutionState{
			Sync:  coprocessor.CoProcParallel,
			Yield: cart.mem.yield,
		}
	}
	return coprocessor.CoProcExecutionState{
		Sync:  coprocessor.CoProcStrongARMFeed,
		Yield: cart.mem.yield,
	}
}

func (cart *Elf) GetCoProc() coprocessor.CartCoProc {
	return cart.arm
}

func (cart *Elf) SetYieldHook(hook coprocessor.CartYieldHook) {
	cart.yieldHook = hook
}
