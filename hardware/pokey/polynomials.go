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

var poly4bit []uint8
var poly5bit []uint8
var poly9bit []uint8
var poly17bit []uint8

type polynomials struct {
	ct4bit  int
	ct5bit  int
	ct9bit  int
	ct17bit int

	// whether to use the 9bit polynomial instead of the 17bit. a pointer to this field is given to
	// each channel. set via the AUDCTL register
	prefer9bit bool

	// from 'Altirra Reference', page 111:
	//
	// "Eight bits of the shift register are visible to the CPU via RANDOM; this is most commonly
	// used for random numbers, but it can also be used to test cycle counting hypotheses. RANDOM
	// shifts right at the rate of one bit per machine cycle. Note that RANDOM reads bits inverted
	// from the shift register itself and the bits seen by the audio circuits"
	//
	// the rnd is updated whenever the 9bit or 17bit polynomial is read
	rnd uint8

	// use the 15Khz clock instead of the 64Khz clock. this is approximately a division of 4
	// 63.9210 / 15.6999 = 4.0714. set via the AUDCTL register
	prefer15Khz bool
}

func (p *polynomials) initialise() {
	p.ct4bit = 0
	p.ct5bit = 0
	p.ct9bit = 0
	p.ct17bit = 0
	p.prefer9bit = false
	p.rnd = 0xff
	p.prefer15Khz = false
}

func (p *polynomials) step() {
	p.ct4bit++
	if p.ct4bit >= len(poly4bit) {
		p.ct4bit = 0
	}

	p.ct5bit++
	if p.ct5bit >= len(poly5bit) {
		p.ct5bit = 0
	}

	p.ct9bit++
	if p.ct9bit >= len(poly9bit) {
		p.ct9bit = 0
	}

	p.ct17bit++
	if p.ct17bit >= len(poly17bit) {
		p.ct17bit = 0
	}

	if p.prefer9bit {
		p.rnd >>= 1
		p.rnd |= poly9bit[p.ct9bit] << 7
	} else {
		p.rnd >>= 1
		p.rnd |= poly17bit[p.ct17bit] << 7
	}
}

func init() {
	// initialisation sequences taken from the Altirra emulator

	var b uint32

	poly4bit = make([]uint8, (1<<4)-1)
	for i := range poly4bit {
		b = (b >> 1) + (^((b << 2) ^ (b << 3)) & 8)
		poly4bit[i] = uint8((b & 1))
	}

	b = 0
	poly5bit = make([]uint8, (1<<5)-1)
	for i := range poly5bit {
		b = (b >> 1) + (^((b << 2) ^ (b << 4)) & 16)
		poly5bit[i] |= uint8((b & 1))
	}

	b = 0
	poly9bit = make([]uint8, (1<<9)-1)
	for i := range poly9bit {
		b = (b >> 1) + (^((b << 8) ^ (b << 3)) & 0x100)
		poly9bit[i] |= uint8((b & 1))
	}

	b = 0
	poly17bit = make([]uint8, (1<<17)-1)
	for i := range poly17bit {
		b = (b >> 1) + (^((b << 16) ^ (b << 11)) & 0x10000)
		poly17bit[i] |= uint8((b >> 8) & 0x01)
	}
}
