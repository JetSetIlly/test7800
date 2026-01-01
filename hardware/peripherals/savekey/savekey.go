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

package savekey

import (
	"unicode"

	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware/peripherals"
	"github.com/jetsetilly/test7800/hardware/peripherals/savekey/i2c"
	"github.com/jetsetilly/test7800/hardware/riot"
	"github.com/jetsetilly/test7800/hardware/tia"
	"github.com/jetsetilly/test7800/logger"
)

type Context interface {
	logger.Permission
}

// SaveKeyState records how incoming signals to the SaveKey will be interpreted.
type SaveKeyState int

// List of valid SaveKeyState values.
const (
	SaveKeyStopped SaveKeyState = iota
	SaveKeyStarting
	SaveKeyAddressHi
	SaveKeyAddressLo
	SaveKeyData
)

// DataDirection indicates the direction of data flow between the console and the SaveKey.
type DataDirection int

// Valid DataDirection values.
const (
	Reading DataDirection = iota
	Writing
)

// SaveKey represents the SaveKey peripheral. It implements the Peripheral
// interface.
type SaveKey struct {
	ctx Context

	portRight bool

	riot peripherals.RIOT
	tia  peripherals.TIA

	// the amount to shift the bits read from swcha
	riotShift int

	// only two bits of the SWCHA value is of interest to the i2c protocol.
	// from the perspective of the second player (in which port the SaveKey is
	// usually inserted) pin 2 is the data signal (SDA) and pin 3 is the
	// clock signal (SCL)
	SDA i2c.Trace
	SCL i2c.Trace

	// incoming data is interpreted depending on the state of the i2c protocol.
	// we also need to know the direction of data flow at any given time and
	// whether the next bit should be acknowledged
	State SaveKeyState
	Dir   DataDirection
	Ack   bool

	// data is sent by the console one bit at a time. see recvBit(), sendBit() and resetBits()
	Bits   uint8
	BitsCt int

	// the core of the SaveKey is an EEPROM.
	EEPROM *EEPROM
}

// NewSaveKey is the preferred method of initialisation for the SaveKey type.
func NewSaveKey(ctx Context, r peripherals.RIOT, t peripherals.TIA, portRight bool) *SaveKey {
	sk := &SaveKey{
		ctx:       ctx,
		portRight: portRight,
		riot:      r,
		tia:       t,
		SDA:       i2c.NewTrace("SDA"),
		SCL:       i2c.NewTrace("SCL"),
		State:     SaveKeyStopped,
		EEPROM:    newEeprom(ctx),
	}

	if portRight {
		sk.riotShift = 4
	} else {
		sk.riotShift = 0
	}

	return sk
}

func (sk *SaveKey) IsAnalogue() bool {
	return false
}

func (sk *SaveKey) IsController() bool {
	return false
}

func (sk *SaveKey) Reset() {
	sk.riot.PortWrite(riot.SWCHA, 0x00>>sk.riotShift, 0x0f<<sk.riotShift)
	if sk.portRight {
		sk.tia.PortWrite(tia.INPT5, 0x80, 0x7f)
	} else {
		sk.tia.PortWrite(tia.INPT4, 0x80, 0x7f)
	}
}

func (sk *SaveKey) Unplug() {
	sk.riot.PortWrite(riot.SWCHA, 0x00>>sk.riotShift, 0x0f<<sk.riotShift)
	if sk.portRight {
		sk.tia.PortWrite(tia.INPT5, 0x00, 0x7f)
	} else {
		sk.tia.PortWrite(tia.INPT4, 0x00, 0x7f)
	}
}

// the active bits in the SWCHA value.
const (
	maskSaveKeySDA = 0b01000000
	maskSaveKeySCL = 0b10000000
)

// the bit sequence to indicate read/write data direction.
const (
	writeSig = 0xa0
	readSig  = 0xa1
)

func (sk *SaveKey) Update(inp gui.Input) error {
	return nil
}

// recvBit will return true if bits field is full. the bits and bitsCt field
// will be reset on the next call.
func (sk *SaveKey) recvBit(v bool) bool {
	if sk.BitsCt >= 8 {
		sk.resetBits()
	}

	if v {
		sk.Bits |= 0x01 << (7 - sk.BitsCt)
	}
	sk.BitsCt++

	return sk.BitsCt == 8
}

// return the next bit in the current byte. end is true if all bits in the
// current byte has been exhausted. next call to sendBit() will use the next
// byte in the EEPROM page.
func (sk *SaveKey) sendBit() (bit bool, end bool) {
	if sk.BitsCt >= 8 {
		sk.resetBits()
	}

	if sk.BitsCt == 0 {
		sk.Bits = sk.EEPROM.get()
	}

	v := (sk.Bits >> (7 - sk.BitsCt)) & 0x01
	bit = v == 0x01
	sk.BitsCt++

	if sk.BitsCt >= 8 {
		end = true
	}

	return bit, end
}

func (sk *SaveKey) resetBits() {
	sk.Bits = 0
	sk.BitsCt = 0
}

func (sk *SaveKey) Tick() {
	swcha, _ := sk.riot.PortRead(riot.SWCHA)
	swcha = (swcha << sk.riotShift) & 0xf0

	// update savekey i2c state
	sk.SDA.Tick(swcha&maskSaveKeySDA == maskSaveKeySDA)
	sk.SCL.Tick(swcha&maskSaveKeySCL == maskSaveKeySCL)

	// check for stop signal before anything else
	if sk.State != SaveKeyStopped && sk.SCL.Hi() && sk.SDA.Rising() {
		logger.Log(sk.ctx, "savekey", "stopped message")
		sk.State = SaveKeyStopped
		sk.EEPROM.Write()
		return
	}

	// if SCL is not changing to a hi state then we don't need to do anything
	if !sk.SCL.Rising() {
		return
	}

	// if the console is waiting for an ACK then handle that now
	if sk.Ack {
		if sk.Dir == Reading && sk.SDA.Falling() {
			sk.riot.PortWrite(riot.SWCHA, maskSaveKeySDA>>sk.riotShift, 0x0f<<sk.riotShift)
			sk.Ack = false
			return
		}
		sk.riot.PortWrite(riot.SWCHA, 0x00>>sk.riotShift, 0x0f<<sk.riotShift)
		sk.Ack = false
		return
	}

	// interpret i2c state depending on which state we are currently in
	switch sk.State {
	case SaveKeyStopped:
		if sk.SDA.Lo() {
			logger.Log(sk.ctx, "savekey", "starting message")
			sk.resetBits()
			sk.State = SaveKeyStarting
		}

	case SaveKeyStarting:
		if sk.recvBit(sk.SDA.Falling()) {
			switch sk.Bits {
			case readSig:
				logger.Log(sk.ctx, "savekey", "reading message")
				sk.resetBits()
				sk.State = SaveKeyData
				sk.Dir = Reading
				sk.Ack = true
			case writeSig:
				logger.Log(sk.ctx, "savekey", "writing message")
				sk.State = SaveKeyAddressHi
				sk.Dir = Writing
				sk.Ack = true
			default:
				logger.Logf(sk.ctx, "savekey", "unrecognised message: %08b", sk.Bits)
				logger.Log(sk.ctx, "savekey", "stopped message")
				sk.State = SaveKeyStopped
			}
		}

	case SaveKeyAddressHi:
		if sk.recvBit(sk.SDA.Falling()) {
			sk.EEPROM.Address = uint16(sk.Bits) << 8
			sk.State = SaveKeyAddressLo
			sk.Ack = true
		}

	case SaveKeyAddressLo:
		if sk.recvBit(sk.SDA.Falling()) {
			sk.EEPROM.Address |= uint16(sk.Bits)
			sk.State = SaveKeyData
			sk.Ack = true

			switch sk.Dir {
			case Reading:
				logger.Logf(sk.ctx, "savekey", "reading from address %#04x", sk.EEPROM.Address)
			case Writing:
				logger.Logf(sk.ctx, "savekey", "writing to address %#04x", sk.EEPROM.Address)
			}
		}

	case SaveKeyData:
		switch sk.Dir {
		case Reading:
			bit, end := sk.sendBit()

			if bit {
				sk.riot.PortWrite(riot.SWCHA, maskSaveKeySDA>>sk.riotShift, 0x0f<<sk.riotShift)
			} else {
				sk.riot.PortWrite(riot.SWCHA, 0x00, 0x0f<<sk.riotShift)
			}

			if end {
				if unicode.IsPrint(rune(sk.Bits)) {
					logger.Logf(sk.ctx, "savekey", "read byte %#02x [%c]", sk.Bits, sk.Bits)
				} else {
					logger.Logf(sk.ctx, "savekey", "read byte %#02x", sk.Bits)
				}
				sk.Ack = true
			}

		case Writing:
			if sk.recvBit(sk.SDA.Falling()) {
				if unicode.IsPrint(rune(sk.Bits)) {
					logger.Logf(sk.ctx, "savekey", "written byte %#02x [%c]", sk.Bits, sk.Bits)
				} else {
					logger.Logf(sk.ctx, "savekey", "written byte %#02x", sk.Bits)
				}
				sk.EEPROM.put(sk.Bits)
				sk.Ack = true
			}
		}
	}
}
