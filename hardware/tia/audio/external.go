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

import "github.com/jetsetilly/test7800/hardware/memory/external"

type ExternalSoundChip interface {
	Label() string
	Step()
	Volume(yield func(uint8))
}

type SoundChipIterator func(func(external.Bus))

func (au *Audio) PiggybackExternalSound(externalChips SoundChipIterator) {
	au.externalChips = au.externalChips[:0]
	if externalChips != nil {
		externalChips(func(bus external.Bus) {
			if sc, ok := bus.(ExternalSoundChip); ok {
				au.externalChips = append(au.externalChips, sc)
			}
		})
	}
}
