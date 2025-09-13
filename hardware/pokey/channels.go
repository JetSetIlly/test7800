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

import "fmt"

type channel struct {
	Registers Registers
	noise     *polynomials

	// the channel number
	num int

	// "Each register controls a divide-by-N counter, where N is the value written to the register
	// AUDF(X), plus 1."
	divCounter uint8

	pulse uint8

	// control of the channel by the main AUDCTL register in the POKEY
	clkMhz bool
	linked bool
	link   *channel

	// reload the divCounter with the current frequency value. this normally happens whenever
	// divCounter reaches 255 (wrap around from zero) but it's slightly different for linked
	// channels
	reload int
}

func (ch *channel) String() string {
	return fmt.Sprintf("Ch%d: %s", ch.num, ch.Registers.String())
}

func (ch *channel) step(clk15Khz, clk64Khz bool) {
	if ch.linked {
		// this early return for a linked channel may cause problems for volume only sample
		// playback. from 'Altirra Reference', page 104
		//
		// "Linking occurs prior to the audio circuitry and thus the waveform settings for the low
		// channel have no effect on the clocking of the high channel. Normally, the low audio
		// channel is muted and only the high channel is used. However, it can also be reused for
		// volume-only effects or even enabled for special effects without affecting the high
		// channel"
		//
		// but we do it because we want to control the channel from the other channel (the one being
		// linked)
		return
	}

	// reload div counter with current frequency value
	if ch.reload > 0 {
		ch.reload--
		if ch.reload == 0 {
			ch.divCounter = ch.Registers.Freq

			// From 'Altirra Reference', page 104
			// "When the high timer underflows, both the low and high timer counters are reloaded together"
			if ch.link != nil {
				ch.link.reload = 7
			}
		}
	}

	// when a channel is linked it is driven by the linked channel. therefore, the hiFreq flag is
	// not relevent to 'this' channel, only to the linked channel
	if ch.link != nil {
		if ch.link.reload > 0 {
			ch.link.reload--
			if ch.link.reload == 0 {
				ch.link.divCounter = ch.link.Registers.Freq
			}
		}

		if !ch.link.clkMhz {
			if !(clk15Khz || clk64Khz) {
				return
			}
		}

		// From 'Altirra Reference', page 104
		//
		// "The automatic reload on underflow is suppressed on the low timer"
		//
		// automatic refers to the comparison with the linked registers frequency register. we
		// therefore return unless divCounter is zero, which indicates that an underflow has
		// occurred naturally
		ch.link.divCounter--
		if ch.link.divCounter != 255 {
			return
		}
	} else if !ch.clkMhz {
		if !(clk15Khz || clk64Khz) {
			return
		}
	}

	// From 'Altirra Reference', page 103
	//
	// "Each channel has an 8-bit countdown timer associated with it to produce clocking pulses. The period for each
	// timer is set by the AUDFx register, specifying a divisor from 1 ($00) to 256 ($FF). The countdown timer produces
	// a pulse each time it underflows and resets, which can then be used to drive an interrupt, the serial port, or sound
	// generation."
	//
	// when the divCounter equals the frequency register the function continues and the next part of
	// the sound is generated
	ch.divCounter--
	if ch.divCounter != 255 {
		return
	}
	ch.reload = 1

	if ch.Registers.Noise&0x01 == 0x01 {
		// "Force Output Volume only"
		ch.pulse = 0x01
	} else {
		switch ch.Registers.Noise & 0x07 {
		case 0x00:
			if ch.noise.prefer9bit {
				ch.pulse = poly9bit[ch.noise.ct9bit]
			} else {
				ch.pulse = poly17bit[ch.noise.ct17bit]
			}
		case 0x02:
			ch.pulse = ch.pulse ^ 0x01
		case 0x04:
			ch.pulse = poly4bit[ch.noise.ct4bit]
		}

		if ch.Registers.Noise&0x08 == 0x00 {
			if poly5bit[ch.noise.ct5bit] != 0x01 {
				return
			}
		}
	}
}

// the actual volume of the channel is the volume in the register multiplied by
// the lower bit of the pulsecounter. this is then used in combination with the
// volume of the other channel to get the actual output volume
func (ch *channel) actualVolume() uint8 {
	return (ch.pulse & 0x01) * ch.Registers.Volume
}
