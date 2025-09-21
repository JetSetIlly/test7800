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

package mix

var softClip [65536]int16

// generate the soft clip curve
func init() {
	for i := -32768; i <= 32767; i++ {
		x := int32(i)

		// saturator y = x / (1 + |x|/32768)
		abs := x
		if abs < 0 {
			abs = -abs
		}
		scale := 32768 + (abs >> 15)
		y := (x * 32767) / scale

		y = max(min(y, 32767), -32768)
		softClip[uint16(i)] = int16(y)
	}
}

// Clip 32bit value so that it doesn't exceed 16bit range
func Clip(x int32) int16 {
	x = max(min(x, 32767), -32768)
	return softClip[uint16(x)]
}
