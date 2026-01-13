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
	"fmt"
	"os"
	"slices"

	"github.com/jetsetilly/test7800/logger"
	"github.com/jetsetilly/test7800/resources"
)

const saveKeyPath = "savekey"

const (
	EEPROMsize     = 0x8000
	EEPROMpageSize = 0x40
	EEPROMnumPages = EEPROMsize / EEPROMpageSize
)

// EEPROM represents the non-volatile memory in the SaveKey peripheral.
type EEPROM struct {
	ctx Context

	// the next address an i2c read/write operation will access
	Address uint16

	// current data
	Data []uint8

	// the data as it is on disk. data is mutable and we need a way of
	// comparing what's on disk with what's in memory.
	Disk []uint8

	// data is dirty and has not been saved to disk
	dirty bool
}

// NewEeprom is the preferred metho of initialisation for the EEPROM type. This
// function will initialise the memory and Read() any existing data from disk.
func newEeprom(ctx Context) *EEPROM {
	ee := &EEPROM{
		ctx:  ctx,
		Data: make([]uint8, EEPROMsize),
		Disk: make([]uint8, EEPROMsize),
	}

	// initialise data with 0xff
	for i := range ee.Data {
		ee.Data[i] = 0xff
	}

	// load from disk
	ee.Restore()

	return ee
}

func (ee *EEPROM) unplug() {
	msg := save(ee.Data)
	logger.Log(ee.ctx, "savekey", msg)
}

func (ee *EEPROM) put(v uint8) {
	ee.Data[ee.Address] = v
	ee.nextAddress()
}

func (ee *EEPROM) get() uint8 {
	defer ee.nextAddress()
	return ee.Data[ee.Address]
}

func (ee *EEPROM) nextAddress() {
	// nextAddress makes sure the address if kept on the same page, by looping back
	// to the start of the current page.
	if ee.Address&0x3f == 0x3f {
		ee.Address ^= 0x3f
	} else {
		ee.Address++
	}
}

// Restore is exposed so that it can be used by an external package - like a GUI package once we add it
func (ee *EEPROM) Restore() {
	d, msg := restore()
	logger.Log(ee.ctx, "savekey", msg)
	if len(d) == 0 {
		ee.dirty = true
		return
	}
	copy(ee.Data, d)
	copy(ee.Disk, d)
}

// Save is exposed so that it can be used by an external package - like a GUI package once we add it
func (ee *EEPROM) Save() {
	if ee.dirty {
		msg := save(ee.Data)
		logger.Log(ee.ctx, "savekey", msg)

		// disk data is now the same as the current data
		ee.dirty = false
		copy(ee.Disk, ee.Data)
	}
}

// IsSaved returns true if disk data is the same as data
func (ee *EEPROM) IsSaved() bool {
	return slices.Equal(ee.Data, ee.Disk)
}

// save returns a string that should be used as a log message
func save(data []uint8) string {
	fn, err := resources.JoinPath(saveKeyPath)
	if err != nil {
		return fmt.Sprintf("could not write eeprom file: %v", err)
	}

	f, err := os.Create(fn)
	if err != nil {
		return fmt.Sprintf("could not write eeprom file: %v", err)
	}

	n, err := f.Write(data)
	if err != nil {
		return fmt.Sprintf("could not write eeprom file: %v", err)
	}

	err = f.Close()
	if err != nil {
		return fmt.Sprintf("could not close eeprom file: %v", err)
	}

	if n != len(data) {
		return fmt.Sprintf("eeprom file has not been truncated during write. %d should be %d", n, len(data))
	}

	return fmt.Sprintf("eeprom file saved to %s", fn)
}

// restore returns a string that should be used as a log message
func restore() ([]uint8, string) {
	fn, err := resources.JoinPath(saveKeyPath)
	if err != nil {
		return []uint8{}, fmt.Sprintf("could not load eeprom file: %v", err)
	}

	f, err := os.Open(fn)
	if err != nil {
		return []uint8{}, fmt.Sprintf("could not load eeprom file: %v", err)
	}
	defer f.Close()

	// get file info. not using Stat() on the file handle because the
	// windows version (when running under wine) does not handle that
	fs, err := os.Stat(fn)
	if err != nil {
		return []uint8{}, fmt.Sprintf("could not load eeprom file: %v", err)
	}

	data := make([]uint8, EEPROMsize)

	if fs.Size() != int64(len(data)) {
		return []uint8{}, fmt.Sprintf("eeprom file is of incorrect length. %d should be %d", fs.Size(), len(data))
	}

	_, err = f.Read(data)
	if err != nil {
		return []uint8{}, fmt.Sprintf("could not load eeprom file: %v", err)
	}

	return data, fmt.Sprintf("eeprom file loaded from %s", fn)
}
