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

package disassembly

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/test7800/hardware/cpu/execution"
	"github.com/jetsetilly/test7800/hardware/cpu/instructions"
)

type Entry struct {
	// copy of the CPU execution. must not be updated except through
	// updateExecutionEntry() function.
	//
	// not that the the Final field of execution.Result may be false is the
	// emulation is stopped mid-execution.
	Result execution.Result

	// the entries below are not defined if Level == EntryLevelUnused

	// string representations of information in execution.Result
	// entry.GetField() will apply white spacing padding suitable for columnation
	Bytecode string
	Address  string
	Operator string
	Operand  string
}

func (e Entry) String() string {
	if e.Result.InInterrupt {
		return fmt.Sprintf("!! %5s %5s %3s %s", e.Bytecode, e.Address, e.Operator, e.Operand)
	}
	return fmt.Sprintf("%5s %5s %3s %s", e.Bytecode, e.Address, e.Operator, e.Operand)
}

// FormatResult creates an Entry for supplied result/bank. It will be assigned
// the specified EntryLevel.
//
// If EntryLevel is EntryLevelExecuted then the disassembly will be updated but
// only if result.Final is true.
func FormatResult(result execution.Result) *Entry {
	e := &Entry{
		Result: result,
	}

	// address of instruction
	e.Address = fmt.Sprintf("$%04x", result.Address)

	// if definition is nil then set the operator field to ??? and return with no further formatting
	if result.Defn == nil {
		e.Operator = "???"
		return e
	}

	// operator of instruction
	e.Operator = result.Defn.Operator.String()

	// bytecode and operand string is assembled depending on the number of
	// expected bytes (result.Defn.Bytes) and the number of bytes read so far
	// (result.ByteCount).
	//
	// the panics cover situations that should never exists. if result
	// validation is active then the panic situations will have been caught
	// then. if validation is not running then the code could theoretically
	// panic but that's okay, they should have been caught in testing.
	switch result.Defn.Bytes {
	case 3:
		switch result.ByteCount {
		case 3:
			operand := result.InstructionData
			e.Operand = fmt.Sprintf("$%04x", operand)
			e.Bytecode = fmt.Sprintf("%02x %02x %02x", result.Defn.OpCode, operand&0x00ff, operand&0xff00>>8)
		case 2:
			operand := result.InstructionData
			e.Operand = fmt.Sprintf("$??%02x", result.InstructionData)
			e.Bytecode = fmt.Sprintf("%02x %02x ?? ", result.Defn.OpCode, operand&0x00ff)
		case 1:
			e.Operand = "$????"
			e.Bytecode = fmt.Sprintf("%02x ?? ??", result.Defn.OpCode)
		case 0:
			panic("this makes no sense. we must have read at least one byte to know how many bytes to expect")
		default:
			panic("we should not be able to read more bytes than the expected number (expected 3)")
		}
	case 2:
		switch result.ByteCount {
		case 2:
			operand := result.InstructionData
			e.Operand = fmt.Sprintf("$%02x", operand)
			e.Bytecode = fmt.Sprintf("%02x %02x", result.Defn.OpCode, operand&0x00ff)
		case 1:
			e.Operand = "$??"
			e.Bytecode = fmt.Sprintf("%02x ??", result.Defn.OpCode)
		case 0:
			panic("this makes no sense. we must have read at least one byte to know how many bytes to expect")
		default:
			panic("we should not be able to read more bytes than the expected number (expected 2)")
		}
	case 1:
		switch result.ByteCount {
		case 1:
			e.Bytecode = fmt.Sprintf("%02x", result.Defn.OpCode)
		case 0:
			panic("this makes no sense. we must have read at least one byte to know how many bytes to expect")
		default:
			panic("we should not be able to read more bytes than the expected number (expected 1)")
		}
	case 0:
		panic("instructions of zero bytes is not possible")
	default:
		panic("instructions of more than 3 bytes is not possible")
	}
	e.Bytecode = strings.TrimSpace(e.Bytecode)

	// decorate operand with addressing mode indicators. this decorates the
	// non-symbolic operand. we also call the decorate function from the
	// Operand() function when a symbol has been found
	e.Operand = addrModeDecoration(e.Operand, e.Result.Defn.AddressingMode)

	return e
}

// add decoration to operand according to the addressing mode of the entry.
// operand taken as an argument because it is called from two different contexts.
func addrModeDecoration(operand string, mode instructions.AddressingMode) string {
	s := operand

	switch mode {
	case instructions.Implied:
	case instructions.Immediate:
		s = fmt.Sprintf("#%s", operand)
	case instructions.Relative:
	case instructions.Absolute:
	case instructions.ZeroPage:
	case instructions.Indirect:
		s = fmt.Sprintf("(%s)", operand)
	case instructions.IndexedIndirect:
		s = fmt.Sprintf("(%s,X)", operand)
	case instructions.IndirectIndexed:
		s = fmt.Sprintf("(%s),Y", operand)
	case instructions.AbsoluteIndexedX:
		s = fmt.Sprintf("%s,X", operand)
	case instructions.AbsoluteIndexedY:
		s = fmt.Sprintf("%s,Y", operand)
	case instructions.ZeroPageIndexedX:
		s = fmt.Sprintf("%s,X", operand)
	case instructions.ZeroPageIndexedY:
		s = fmt.Sprintf("%s,Y", operand)
	}

	return s
}
