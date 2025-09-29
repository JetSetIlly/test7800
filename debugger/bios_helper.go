package debugger

import (
	"fmt"

	"github.com/jetsetilly/test7800/hardware/cpu"
)

type biosHelper struct {
	// some cartridge types will bypass the BIOS. it's possible to force the
	// BIOS to be skipped with this flag in all cases
	bypass bool

	// checksum has passed. there is no need to continue checking for a fail condition
	cartridgeAccepted bool

	// set to false to ignore checksum failure in supported BIOS'
	checksum bool
}

func (hlp *biosHelper) reset(md5sum string) {
	const supportedBIOS = "0x0763f1ffb006ddbe32e52d497ee848ae"
	hlp.cartridgeAccepted = hlp.bypass && md5sum == supportedBIOS
}

func (hlp *biosHelper) cartridgeAcceptedCheck(mc *cpu.CPU) error {
	if hlp.cartridgeAccepted {
		return nil
	}
	if mc.PC.Address() == 0x26c2 {
		if hlp.checksum {
			return fmt.Errorf("checksum fail. ROM not signed for BIOS")
		}
		mc.PC.Load(0x23f9)
	}
	if mc.PC.Address() == 0x23f9 {
		hlp.cartridgeAccepted = true
	}
	return nil
}
