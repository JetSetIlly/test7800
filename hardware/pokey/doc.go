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

// Package pokey implements the audio generation of the POKEY. It is based on
// the work for TIA audio, implemented elsewhere in Test7800.
//
// Unlike TIA audio it is not intended to be stepped directly from the main
// console loop. Rather, it piggybacks on the TIA and is ticked in lock-step
// with the TIA.
//
// Information about POKEY specifically is taken from:
//
// https://7800.8bitdev.org/index.php/POKEY_C012294_Documentation
//
// This document will be referred to as "POKEY_C012294" in any code comments in
// this package. And quotations in the comments should be assumed to be from
// this document.
//
// Also used as a reference is chapter 5 of the Altirra Hardware Reference
// Manual:
//
// https://www.virtualdub.org/downloads/Altirra%20Hardware%20Reference%20Manual.pdf
//
// References to this document in comments will be abbreviated to 'Altirra Reference'
//
// Also used as a reference for POKEY is the relevant parts of the Altirra
// emulator (specifically v4.31)
//
// https://www.virtualdub.org/altirra.html
//
// And finally, the original POKEY document from Atari for additional clarity:
//
// http://visual6502.org/images/C012294_Pokey/pokey.pdf
//
// References in comments will be abbreviated to 'Atari POKEY'
package pokey
