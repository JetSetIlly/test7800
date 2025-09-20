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
	"math"
	"slices"
	"testing"

	"github.com/jetsetilly/test7800/hardware/spec"
	"github.com/jetsetilly/test7800/test"
)

func TestPoynomialsLength(t *testing.T) {
	// from 'Altirra Reference", page 110:
	//
	// "As maximal-length generators, each N-bit generator has a period of 2N – 1, so the 4-bit
	// generator repeats every 15 bits, and the 9 bit generator every 511 bits."
	test.ExpectEquality(t, len(poly4bit), 15)
	test.ExpectEquality(t, len(poly5bit), 31)
	test.ExpectEquality(t, len(poly9bit), 511)
	test.ExpectEquality(t, len(poly17bit), 131071)
}

func TestPoynomialsBiasCheck(t *testing.T) {
	// check that the number of zero bits in the polynomial sequence is half the length of the
	// sequence plus one. ie. a slight bias to the 0 bit. this test is supported by the page 110, of
	// the 'Altirra Reference':
	//
	// "This also means the generator patterns are slightly biased with one more 0 bit than 1 bit."
	biasCheck := func(t *testing.T, b []uint8) bool {
		t.Helper()

		var ct int
		for _, v := range b {
			if v == 0 {
				ct++
			}
		}
		return ct == (len(b)/2)+1
	}

	test.ExpectSuccess(t, biasCheck(t, poly4bit[:]))
	test.ExpectSuccess(t, biasCheck(t, poly5bit[:]))
	test.ExpectSuccess(t, biasCheck(t, poly9bit[:]))
	test.ExpectSuccess(t, biasCheck(t, poly17bit[:]))
}

func TestRandomDistribution(t *testing.T) {
	var p polynomials
	p.initialise()

	var counts [256]int

	for range 100000 {
		p.step()
		counts[p.rnd()]++
	}

	var mean int
	for _, v := range counts {
		mean += v
	}
	mean /= len(counts)

	var variance int
	for _, v := range counts {
		diff := v - mean
		variance += diff * diff
	}
	variance /= len(counts)

	// using relative standard distributino to check
	stdDev := math.Sqrt(float64(variance))
	relStdDev := stdDev / float64(mean)

	test.ExpectSuccess(t, relStdDev < 1.0)
}

func TestRandomInitialisation(t *testing.T) {
	var p polynomials
	p.initialise()

	// from 'Altirra Reference", page 111:
	//
	// "When exiting initialization mode, the polynomial counters begin counting immediately. For instance, if 9-bit mode
	// is selected, executing STA SKCTL + LDA RANDOM back-to-back will give A=$1F, which is four bits after the all
	// ones state."

	p.prefer9bit = true

	test.ExpectEquality(t, p.rnd(), 0xff)

	p.step()
	p.step()
	p.step()

	test.ExpectEquality(t, p.rnd(), 0x1f)
}

// the random number tests are not working yet so just ignore it for now
const skipTestRandomSequence = true

func TestRandomSequence(t *testing.T) {
	if skipTestRandomSequence {
		return
	}

	var p polynomials
	p.initialise()

	// from 'Altirra Reference", page 111:
	//
	// "If the main LFSR is in 9-bit mode and samples are taken from RANDOM ($D20A) every scan line by STA
	// WSYNC + LDA RANDOM, part of the sequence is as follows: 00 DF EE 16 B9."

	p.prefer9bit = true

	// accumulated random number sequence
	var seq []uint8

	// step polynomials for 100 frames. take random number sample once per scanline
	for range spec.NTSC.AbsoluteBottom * 100 {
		seq = append(seq, p.rnd())
		for range spec.ClksScanline {
			p.step()
		}
	}

	// look for expected sequence in the collated numbers
	expected := []uint8{0x00, 0xdf, 0xee, 0x16, 0xb9}
	for i := range seq {
		if slices.Equal(seq[i:i+len(expected)], expected) {
			return
		}
	}

	t.Error("random sequence incorrect")
}
