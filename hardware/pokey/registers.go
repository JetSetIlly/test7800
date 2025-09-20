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

package pokey

import (
	"fmt"
)

type Registers struct {
	// noise and volume come from the PAUDCx registers. noise is the upper 4-bits and the volume is
	// the lower 4-bits
	Noise  uint8
	Volume uint8

	// frequency value comes from the PAUDFx registers
	Freq uint8
}

func (reg Registers) String() string {
	return fmt.Sprintf("%04b @ %05b ^ %04b", reg.Noise, reg.Freq, reg.Volume)
}

func (pk *Pokey) Access(write bool, idx uint16, data uint8) (uint8, bool, error) {
	if write {
		switch idx {
		case 0x00 + pk.origin: // PAUDFO
			pk.channel[0].Registers.Freq = data
		case 0x01 + pk.origin: // PAUDCO
			pk.channel[0].Registers.Noise = (data & 0xf0) >> 4
			pk.channel[0].Registers.Volume = (data & 0x0f)
			pk.channel[0].predetermineAUDC()
		case 0x02 + pk.origin: // PAUDF1
			pk.channel[1].Registers.Freq = data
		case 0x03 + pk.origin: // PAUDC1
			pk.channel[1].Registers.Noise = (data & 0xf0) >> 4
			pk.channel[1].Registers.Volume = (data & 0x0f)
			pk.channel[1].predetermineAUDC()
		case 0x04 + pk.origin: // PAUDF2
			pk.channel[2].Registers.Freq = data
		case 0x05 + pk.origin: // PAUDC2
			pk.channel[2].Registers.Noise = (data & 0xf0) >> 4
			pk.channel[2].Registers.Volume = (data & 0x0f)
			pk.channel[2].predetermineAUDC()
		case 0x06 + pk.origin: // PAUDF3
			pk.channel[3].Registers.Freq = data
		case 0x07 + pk.origin: // PAUDC3
			pk.channel[3].Registers.Noise = (data & 0xf0) >> 4
			pk.channel[3].Registers.Volume = (data & 0x0f)
			pk.channel[3].predetermineAUDC()

		case 0x08 + pk.origin: // PAUDCTRL
			pk.noise.prefer9bit = data&0x80 == 0x80
			pk.channel[0].clkMhz = data&0x40 == 0x40
			pk.channel[2].clkMhz = data&0x20 == 0x20

			if data&0x10 == 0x10 {
				pk.channel[1].lnk16High = &pk.channel[0]
				pk.channel[0].lnk16Low = true
			} else {
				pk.channel[1].lnk16High = nil
				pk.channel[0].lnk16Low = false
			}
			if data&0x08 == 0x08 {
				pk.channel[3].lnk16High = &pk.channel[2]
				pk.channel[2].lnk16Low = true
			} else {
				pk.channel[3].lnk16High = nil
				pk.channel[2].lnk16Low = false
			}

			if data&0x04 == 0x04 {
				pk.channel[2].lnkFilter = &pk.channel[0]
			} else {
				pk.channel[2].lnkFilter = nil
				pk.channel[0].filter = 0x01
			}
			if data&0x02 == 0x02 {
				pk.channel[3].lnkFilter = &pk.channel[1]
			} else {
				pk.channel[3].lnkFilter = nil
				pk.channel[1].filter = 0x01
			}

			pk.prefer15Khz = data&0x01 == 0x01
		case 0x09 + pk.origin:
		case 0x0a + pk.origin:
		case 0x0b + pk.origin:
		case 0x0c + pk.origin:
		case 0x0d + pk.origin:
		case 0x0e + pk.origin:
		case 0x0f + pk.origin: // SKCTL
			// we're only interested in the reset bits of SKCTL
			pk.initState = data&0x03 == 0x00
		default:
			return 0, false, nil
		}

		return 0, true, nil
	}

	switch idx {
	case 0x01 + pk.origin:
		return 0, true, nil
	case 0x02 + pk.origin:
		return 0, true, nil
	case 0x03 + pk.origin:
		return 0, true, nil
	case 0x04 + pk.origin:
		return 0, true, nil
	case 0x05 + pk.origin:
		return 0, true, nil
	case 0x06 + pk.origin:
		return 0, true, nil
	case 0x07 + pk.origin:
		return 0, true, nil
	case 0x08 + pk.origin:
		return 0, true, nil
	case 0x09 + pk.origin:
		return 0, true, nil
	case 0x0a + pk.origin: // RANDOM
		if pk.initState {
			return 0xff, true, nil
		}
		return pk.noise.rnd(), true, nil
	case 0x0b + pk.origin:
		return 0, true, nil
	case 0x0c + pk.origin:
		return 0, true, nil
	case 0x0d + pk.origin:
		return 0, true, nil
	case 0x0e + pk.origin:
		return 0, true, nil
	case 0x0f + pk.origin:
		return 0, true, nil
	}

	return 0, false, nil
}
