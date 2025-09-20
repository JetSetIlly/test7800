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
	"strings"

	"github.com/jetsetilly/test7800/hardware/spec"
	"github.com/jetsetilly/test7800/hardware/tia/audio/mix"
)

// The TIA emulation takes two samples per scanline, so by definition the sample
// frequency is double the horizontal scan rate for the machine
//
// 31468.52 for NTSC
// 31250 for PAL
//
// an average of the two can be rounded to 31360
const AverageSampleFreq = 31360

// For a long time we used a sample frequency of 31400, or 15700*2
const OldSampleFreq = 31400

// The TIA audio volume state for both channles is sampled twice per scanline
const SamplesPerScanline = 2

// Audio is the implementation of the TIA audio sub-system
type Audio struct {
	// the reference frequency for all sound produced by the TIA is 30Khz.
	// this is the 3.58Mhz clock, which the TIA operates at, divided by
	// 114. that's one half of a scanline so we count to 228 and update
	// twice in that time
	clock int

	// the volume is sampled every colour clock and the volume at each clock is
	// summed. at fixed points, the volume is averaged
	sampleSum   []int
	sampleSumCt int

	// From the "Stella Programmer's Guide":
	//
	// "There are two audio circuits for generating sound. They are identical but
	// completely independent and can be operated simultaneously [...]"
	Channel0 channel
	Channel1 channel

	// the volume output for each channel
	vol0 uint8
	vol1 uint8

	// any chips in the external device that provide sound
	externalChips []ExternalSoundChip
}

// NewAudio is the preferred method of initialisation for the Audio sub-system.
func NewAudio() *Audio {
	au := &Audio{
		sampleSum: make([]int, 2),
	}
	return au
}

// Snapshot creates a copy of the TIA Audio sub-system in its current state.
func (au *Audio) Snapshot() *Audio {
	n := *au
	return &n
}

func (au *Audio) String() string {
	s := strings.Builder{}
	s.WriteString("ch0: ")
	s.WriteString(au.Channel0.String())
	s.WriteString("  ch1: ")
	s.WriteString(au.Channel1.String())
	return s.String()
}

// UpdateTracker changes the state of the attached tracker. Should be called
// whenever any of the audio registers have changed.
func (au *Audio) UpdateTracker() {
}

func (au *Audio) Step() bool {
	var changed bool

	// sum volume bits
	au.sampleSum[0] += int(au.Channel0.actualVolume())
	au.sampleSum[1] += int(au.Channel1.actualVolume())
	au.sampleSumCt++

	if (au.clock >= 8 && au.clock <= 11) || (au.clock >= 80 && au.clock <= 83) {
		au.Channel0.phase0()
		au.Channel1.phase0()
	} else if (au.clock >= 36 && au.clock <= 39) || (au.clock >= 148 && au.clock <= 151) {
		au.Channel0.phase1()
		au.Channel1.phase1()

		// take average of sum of volume bits
		au.vol0 = uint8(au.sampleSum[0]/au.sampleSumCt) & 0x0f
		au.vol1 = uint8(au.sampleSum[1]/au.sampleSumCt) & 0x0f
		au.sampleSum[0] = 0
		au.sampleSum[1] = 0
		au.sampleSumCt = 0

		changed = true
	}

	au.clock += 4
	if au.clock >= spec.ClksScanline {
		au.clock -= spec.ClksScanline
	}

	// step external soundchips at the base rate of the machine
	for _, ch := range au.externalChips {
		ch.Step()
	}

	return changed
}

// Mono returns the mixed volume from all audio sources
func (au *Audio) Mono() int16 {
	sum := mix.Mono(au.vol0, au.vol1)

	for _, xc := range au.externalChips {
		xc.Volume(func(v uint8) {
			sum += int16(v) << 8
		})
	}

	return sum
}

func (au *Audio) Stereo() (int16, int16) {
	ch1 := int16(au.vol0) << 8
	ch2 := int16(au.vol1) << 8

	switch len(au.externalChips) {
	case 0:
	case 1:
		for _, xc := range au.externalChips {
			xc.Volume(func(v uint8) {
				ch1 += int16(v) << 8
				ch2 += int16(v) << 8
			})
		}
	default:
		for i, xc := range au.externalChips {
			xc.Volume(func(v uint8) {
				if i&0x01 == 0x01 {
					ch1 += int16(v) << 8
				} else {
					ch2 += int16(v) << 8
				}
			})
		}
	}

	return ch1, ch2
}
