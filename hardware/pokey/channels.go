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
	lnk16Low     *channel
	lnk16High    *channel
	lnk16HighClk bool

	// two channels can be linked to create a high-pass filter
	//
	// the resting value of filter should be 0x01 for channels 0 and 1 if they are not being linked
	// to (ie. being filtered) by another channel. for channels 2 and 3 the filter value should
	// always be 0x00. the lnkFilter field should always be nil for channels 0 and 1
	//
	// from 'Altirra Reference', page 107:
	//
	// "When the high-pass filter is disabled, the high-pass flip-flop is forced to a 1, but the XOR
	// still takes place. This causes the digital output from channels 1 and 2 to be inverted"
	//
	// (the channel in which lnkFilter is not nil is the channel doing the filtering. the channel
	// being pointed to by lnkFilter is being filtered)
	lnkFilter *channel
	filter    uint8

	lnk2Tone *channel

	// another channel can affect the final value of the pulse field by flipping the xor field. this
	// creates a high-pass filter on the filtered channel

	// reload the divCounter with the current frequency value. this normally happens whenever
	// divCounter reaches 255 (wrap around from zero) but it's slightly different for linked
	// channels
	reload int

	// predetermined flags based on the current noise value. the pure, poly4 and poly5 flags are
	// mutually exclusive. if all of those flags are false then the channel is in poly17 mode, or
	// poly9 mode if the prefer9bit flags is enabled in the polynomials field
	modePure  bool
	modePoly4 bool
	modePoly5 bool

	// volume-only mode, predetermined from the current noise value. this directly affects the
	// volume returned by the actualVolume() function, it is quite distinct to the other predecoded
	// values
	modeVolumeOnly bool
}

func (ch *channel) String() string {
	return fmt.Sprintf("Ch%d: %s", ch.num, ch.Registers.String())
}

func (ch *channel) loadAUDF(data uint8) {
	// current divCounter continues as normal even though we've changed the frequency
	// in the register
	ch.Registers.Freq = data
}

func (ch *channel) loadAUDC(data uint8) {
	ch.Registers.Noise = (data & 0xf0) >> 4
	ch.Registers.Volume = (data & 0x0f)
	ch.modePure = ch.Registers.Noise&0x07 == 0x02
	ch.modePoly4 = ch.Registers.Noise&0x07 == 0x04
	ch.modePoly5 = ch.Registers.Noise&0x08 != 0x08
	ch.modeVolumeOnly = ch.Registers.Noise&0x01 == 0x01
}

func (ch *channel) isLnk16High() bool {
	return ch.lnk16Low != nil
}

func (ch *channel) isLnk16Low() bool {
	return ch.lnk16High != nil
}

func (ch *channel) step(clk bool) {
	if ch.isLnk16High() {
		if !ch.lnk16HighClk {
			// from 'Altirra Reference', page 104
			//
			// "Linking occurs prior to the audio circuitry and thus the waveform settings for the low
			// channel have no effect on the clocking of the high channel. Normally, the low audio
			// channel is muted and only the high channel is used. However, it can also be reused for
			// volume-only effects or even enabled for special effects without affecting the high
			// channel"
			return
		}
		ch.lnk16HighClk = false
	} else if !ch.clkMhz && !clk {
		return
	}

	// reload div counter with current frequency value
	if ch.reload > 0 {
		ch.reload--
		if ch.reload == 0 {
			ch.divCounter = ch.Registers.Freq

			// the filter on the linked channel is flipped when this channel expires/reloads. the output
			// of this channel (ie. the phase) has no effect. from 'Altirra Reference', page 107
			//
			// "None of the AUDC3/4 bits on the high channel affect high-pass operation"
			if ch.lnkFilter != nil {
				ch.lnkFilter.filter = ^ch.lnkFilter.pulse
			}

			if ch.lnk2Tone != nil {
				ch.lnk2Tone.reload = 1
				return
			}
		}
	}

	// from 'Altirra Reference', page 103
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

	// how we reload the divCounter depends on if the channel is part of a 16bit timer; and if it is,
	// which half the timer the channel it is representing
	if ch.isLnk16Low() {
		ch.lnk16High.lnk16HighClk = true
	} else {
		ch.reload = 1
		if ch.isLnk16High() {
			ch.lnk16Low.reload = 7
		}
	}

	if ch.modePoly5 {
		if poly5bit[ch.noise.ct5bit] != 0x01 {
			return
		}
	}

	if ch.modePure {
		ch.pulse = ch.pulse ^ 0x01
	} else if ch.modePoly4 {
		ch.pulse = poly4bit[ch.noise.ct4bit]
	} else {
		if ch.noise.prefer9bit {
			ch.pulse = poly9bit[ch.noise.ct9bit]
		} else {
			ch.pulse = poly17bit[ch.noise.ct17bit]
		}
	}
}

// the actual volume of the channel is the volume in the register multiplied by the lower bit of the
// pulsecounter. this is then used in combination with the volume of the other channel to get the
// actual output volume
func (ch *channel) actualVolume() uint8 {
	// from "Altirra Reference", page 105
	//
	// "Bit 4 enables volume-only mode. When set, the waveform output is overridden and hardwired on at the output.
	// None of the other distortion bits affect the audio output in this mode, though they still do affect hidden state in the
	// audio circuitry, as the clocking and noise circuits still run but just don’t have an effect on the audio output."
	if ch.modeVolumeOnly {
		return ch.Registers.Volume
	}
	return ((ch.pulse ^ ch.filter) & 0x01) * ch.Registers.Volume
}
