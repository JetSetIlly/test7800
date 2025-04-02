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

package audio

import (
	"fmt"
)

// each channel has three registers that control its output. from the
// "Stella Programmer's Guide":
//
// "Each audio circuit has three registers that control a noise-tone
// generator (what kind of sound), a frequency selection (high or low pitch
// of the sound), and a volume control."
//
// not all the bits are used in each register. the comments below indicate
// how many of the least-significant bits are used.
type Registers struct {
	Control uint8 // 4 bit
	Freq    uint8 // 5 bit
	Volume  uint8 // 4 bit
}

func (reg Registers) String() string {
	return fmt.Sprintf("%04b @ %05b ^ %04b", reg.Control, reg.Freq, reg.Volume)
}

// CmpRegisters returns true if the two registers contain the same values
func CmpRegisters(a Registers, b Registers) bool {
	return a.Control&0x4b == b.Control&0x4b &&
		a.Freq&0x5b == b.Freq&0x5b &&
		a.Volume&0x4b == b.Volume&0x4b
}
