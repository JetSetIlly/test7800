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

package elf

func vcsInjectDmaData(mem *elfMemory) {
	// vcsInjectDmaData() cannot be streamed so we disable streaming from here on in
	if mem.stream.active {
		mem.endStrongArmFunction()
		mem.stream.startDrain()
		mem.stream.active = false
		return
	}

	// __attribute__((long_call, section(".RamFunc")))
	// void vcsInjectDmaData(uint16_t address, uint8_t count, const uint8_t* pBuffer)
	// {
	// 	DATA_OUT = pBuffer[0];
	// 	while(ADDR_IN != address)
	// 		;
	// 	SET_DATA_MODE_OUT;
	// 	for(int i = 1; i < count; i++){
	// 		address++;
	// 		while(ADDR_IN != address)
	// 			;
	// 		DATA_OUT = pBuffer[i];
	// 	}
	// 	while(ADDR_IN & 0x1000)
	// 		;
	// 	SET_DATA_MODE_IN;
	// }

	address := uint16(mem.strongarm.running.registers[0])
	count := uint8(mem.strongarm.running.registers[1])
	buffer := mem.strongarm.running.registers[2]

	switch mem.strongarm.running.state {
	case 0:
		mem.strongarm.running.counter = 0
		mem.strongarm.running.state++
	case 1:
		if mem.strongarm.running.counter >= int(count) {
			mem.endStrongArmFunction()
		} else {
			data, origin := mem.MapAddress(buffer, false, false)
			mem.strongarm.nextRomAddress = address + uint16(mem.strongarm.running.counter)
			if mem.injectRomByte(uint8((*data)[buffer-origin+uint32(mem.strongarm.running.counter)])) {
				mem.strongarm.running.counter++
			}
		}
	}
}

// void vcsWaitForAddress(uint16_t address)
func vcsWaitForAddress(mem *elfMemory) {
	addrIn := uint16(mem.gpio.data[ADDR_IDR])
	addrIn |= uint16(mem.gpio.data[ADDR_IDR+1]) << 8
	addrIn &= Memtop
	address := uint16(mem.strongarm.running.registers[0])
	if addrIn == address {
		mem.endStrongArmFunction()
	}
}

func vcsSnoopRead(mem *elfMemory) {
	// vcsSnoopRead() cannot be streamed so we disable streaming from here on in
	if mem.stream.active {
		mem.endStrongArmFunction()
		mem.stream.startDrain()
		mem.stream.active = false
		return
	}

	// __attribute__((long_call, section(".RamFunc")))
	// uint8_t vcsSnoopRead(uint16_t address)
	// {
	// 	uint8_t result = 0xff;
	// 	while(ADDR_IN != address)
	// 		;
	// 	while(ADDR_IN == address)
	// 		result = DATA_IN;
	// 	return result;
	// }

	switch mem.strongarm.running.state {
	case 0:
		address := uint16(mem.strongarm.running.registers[0])
		addrIn := uint16(mem.gpio.data[ADDR_IDR])
		addrIn |= uint16(mem.gpio.data[ADDR_IDR+1]) << 8
		addrIn &= Memtop
		if addrIn == address {
			mem.strongarm.running.state++
		}
	case 1:
		mem.arm.RegisterSet(0, uint32(mem.gpio.data[DATA_IDR]))
		mem.endStrongArmFunction()
	}
}
