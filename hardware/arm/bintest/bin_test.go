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

package binarytesting_test

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/jetsetilly/test7800/coprocessor"
	"github.com/jetsetilly/test7800/hardware/arm"
	"github.com/jetsetilly/test7800/hardware/arm/architecture"
	"github.com/jetsetilly/test7800/test"
)

type logger struct {
	output []byte
}

func (l *logger) Write(p []byte) (n int, err error) {
	l.output = append(l.output, p...)
	return len(p), nil
}

func (l *logger) Dump() {
	fmt.Println(string(l.output))
}

type registers interface {
	String() string
	Register(reg int) (uint32, bool)
}

type disasm struct {
	log *logger
	arm registers
}

func (d *disasm) printTouchedRegs(operand string) {
	var printed bool

	// end on newline only if we've printed something
	defer func() {
		if printed {
			fmt.Fprintln(d.log)
		}
		fmt.Fprintln(d.log, d.arm.String())
	}()

	for _, op := range strings.Split(operand, ",") {
		op = strings.TrimSpace(op)
		op = strings.ToUpper(op)

		if strings.HasPrefix(op, "R") {
			reg, err := strconv.Atoi(strings.TrimLeft(op, "R"))
			if err != nil {
				return
			}
			v, ok := d.arm.Register(reg)
			if !ok {
				return
			}
			fmt.Fprintf(d.log, "\tR%d=%08x (%d)", reg, v, v)
			printed = true
		}

		if strings.HasPrefix(op, "S") {
			reg, err := strconv.Atoi(strings.TrimLeft(op, "S"))
			if err != nil {
				return
			}
			v, ok := d.arm.Register(reg + 64) // FPU register S0 has absolute register of 64
			if !ok {
				return
			}
			fmt.Fprintf(d.log, "\tS%d=%08x (%d)", reg, v, v)
			printed = true
		}
	}
}

// Start is called at the beginning of coprocessor program execution.
func (d *disasm) Start() {
}

// Step called after every instruction in the coprocessor program.
func (d *disasm) Step(e coprocessor.CartCoProcDisasmEntry) {
	if e == nil {
		return
	}

	ae := e.(arm.DisasmEntry)
	if ae.Is32bit {
		fmt.Fprintf(d.log, "%s %04x %04x %s %s\n", ae.Address, ae.OpcodeHi, ae.Opcode, ae.Operator, ae.Operand)
	} else {
		fmt.Fprintf(d.log, "%s      %04x %s %s\n", ae.Address, ae.Opcode, ae.Operator, ae.Operand)
	}

	d.printTouchedRegs(ae.Operand)
	fmt.Fprintln(d.log)
}

// End is called when coprocessor program has finished.
func (d *disasm) End(s coprocessor.CartCoProcDisasmSummary) {
	fmt.Fprintln(d.log, d.arm.String())
}

type memory struct {
	disasm disasm

	data       []byte
	dataOrigin uint32
	dataMemtop uint32

	stack       []byte
	stackOrigin uint32
	stackMemtop uint32
}

func (mem *memory) MapAddress(addr uint32, write bool, executing bool) (*[]byte, uint32) {
	if addr >= mem.dataOrigin && addr <= mem.dataMemtop {
		return &mem.data, mem.dataOrigin
	}
	if addr >= mem.stackOrigin && addr <= mem.stackMemtop {
		return &mem.stack, mem.stackOrigin
	}
	if addr == 0xffffffff {
		return &mem.data, mem.dataOrigin
	}
	return nil, 0
}

// Return reset addreses for the Stack Pointer register; the Link Register;
// and Program Counter
func (mem *memory) ResetVectors() (uint32, uint32, uint32) {
	return mem.stackMemtop - 0x04, 0x00, 0x00
}

// Return true is address contains executable instructions.
func (mem *memory) IsExecutable(addr uint32) bool {
	return true
}

func (mem *memory) ARMinterrupt(addr uint32, val1 uint32, val2 uint32) (arm.ARMinterruptReturn, error) {
	return arm.ARMinterruptReturn{}, nil
}

func run(t *testing.T, name string, testBin []byte) {
	mem := memory{
		disasm: disasm{
			log: &logger{},
		},
		data:        testBin,
		dataOrigin:  0,
		stack:       make([]byte, 0x2000),
		stackOrigin: 0x1ffff000,
	}
	mem.dataMemtop = mem.dataOrigin + uint32(len(mem.data))
	mem.stackMemtop = mem.stackOrigin + uint32(len(mem.stack))

	mc := arm.NewARM(architecture.NewMap("PlusCart"), &mem, &mem)

	mem.disasm.arm = mc
	mc.SetDisassembler(&mem.disasm)

	fmt.Fprintln(mem.disasm.log, name)
	yld, _ := mc.Run()
	if !test.ExpectEquality(t, yld.Type, coprocessor.YieldSyncWithVCS) {
		t.Fail()
	}
	result, _ := mc.Register(0)
	if !test.ExpectEquality(t, result, 0x00) {
		mem.disasm.log.Dump()
	}
}

//go:embed "fpu/test_1/test.bin"
var fpuTestBin1 []byte

//go:embed "fpu/test_2/test.bin"
var fpuTestBin2 []byte

//go:embed "fpu/test_3/test.bin"
var fpuTestBin3 []byte

//go:embed "fpu/test_4/test.bin"
var fpuTestBin4 []byte

//go:embed "fpu/test_5/test.bin"
var fpuTestBin5 []byte

//go:embed "fpu/test_6/test.bin"
var fpuTestBin6 []byte

//go:embed "fpu/test_7/test.bin"
var fpuTestBin7 []byte

//go:embed "fpu/test_8/test.bin"
var fpuTestBin8 []byte

func TestAllBinaries(t *testing.T) {
	run(t, "fpuTestBin1", fpuTestBin1)
	run(t, "fpuTestBin2", fpuTestBin2)
	run(t, "fpuTestBin3", fpuTestBin3)
	run(t, "fpuTestBin4", fpuTestBin4)
	run(t, "fpuTestBin5", fpuTestBin5)
	run(t, "fpuTestBin6", fpuTestBin6)
	run(t, "fpuTestBin7", fpuTestBin7)
	run(t, "fpuTestBin8", fpuTestBin8)
}
