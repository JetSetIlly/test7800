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

	// pulse controls the output of the actualVolume() function. it has no effect if the channel is
	// in volume-only mode
	pulse uint8

	// clock preference for the channel
	clkMhz bool

	// two channels can be linked to create a 16bit timer
	//
	// if lnk16Low is true then the channel is the low-byte of the timer. if lnk16High is not nil
	// then the channel is the high-byte of the timer (the channel being pointed to is the
	// low-byte). both fields should not be 'true' at the same time
	//
	// channels 0 and 2 can only ever be the low-byte. and channels 1 and 3 can only ever be the
	// high-byte
	lnk16Low  bool
	lnk16High *channel

	// two channels can be linked to create a high-pass filter
	//
	// the resting value of filter should be 0x01 for channels 0 and 1 if they are not being linked
	// to (ie. being filtered) by another channel. for channels 2 and 3 the filter value should
	// always be 0x00. the lnkFilter field should always be nil for channels 0 and 1
	//
	// From 'Altirra Reference', page 107:
	//
	// "When the high-pass filter is disabled, the high-pass flip-flop is forced to a 1, but the XOR
	// still takes place. This causes the digital output from channels 1 and 2 to be inverted"
	lnkFilter *channel
	filter    uint8

	// another channel can affect the final value of the pulse field by flipping the xor field. this
	// creates a high-pass filter on the filtered channel

	// reload the divCounter with the current frequency value. this normally happens whenever
	// divCounter reaches 255 (wrap around from zero) but it's slightly different for linked
	// channels
	reload int
}

func (ch *channel) String() string {
	return fmt.Sprintf("Ch%d: %s", ch.num, ch.Registers.String())
}

func (ch *channel) step(clk15Khz, clk64Khz bool) {
	if ch.lnk16Low {
		// this early return for a linked channel may cause problems for volume only sample
		// playback. from 'Altirra Reference', page 104
		//
		// "Linking occurs prior to the audio circuitry and thus the waveform settings for the low
		// channel have no effect on the clocking of the high channel. Normally, the low audio
		// channel is muted and only the high channel is used. However, it can also be reused for
		// volume-only effects or even enabled for special effects without affecting the high
		// channel"
		//
		// but we do it because we want to control the channel from the other channel
		return
	}

	// reload div counter with current frequency value
	if ch.reload > 0 {
		ch.reload--
		if ch.reload == 0 {
			ch.divCounter = ch.Registers.Freq

			// From 'Altirra Reference', page 104
			// "When the high timer underflows, both the low and high timer counters are reloaded together"
			if ch.lnk16High != nil {
				ch.lnk16High.reload = 7
			}

			if ch.lnkFilter != nil {
				ch.lnkFilter.filter = ch.lnkFilter.filter ^ 0x01
			}
		}
	}

	// when a channel is linked it is driven by the channel specified in the link field. therefore,
	// the hiFreq flag is not relevent to the current channel, only to the other channel
	if ch.lnk16High != nil {
		if ch.lnk16High.reload > 0 {
			ch.lnk16High.reload--
			if ch.lnk16High.reload == 0 {
				ch.lnk16High.divCounter = ch.lnk16High.Registers.Freq
			}
		}

		if !ch.lnk16High.clkMhz {
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
		ch.lnk16High.divCounter--
		if ch.lnk16High.divCounter != 255 {
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

// the actual volume of the channel is the volume in the register multiplied by
// the lower bit of the pulsecounter. this is then used in combination with the
// volume of the other channel to get the actual output volume
func (ch *channel) actualVolume() uint8 {
	// From "Altirra Reference", page 105
	//
	// "Bit 4 enables volume-only mode. When set, the waveform output is overridden and hardwired on at the output.
	// None of the other distortion bits affect the audio output in this mode, though they still do affect hidden state in the
	// audio circuitry, as the clocking and noise circuits still run but just don’t have an effect on the audio output."
	if ch.Registers.Noise&0x01 == 0x01 {
		return ch.Registers.Volume
	}
	return ((ch.pulse ^ ch.filter) & 0x01) * ch.Registers.Volume
}
