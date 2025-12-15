package debugger

import (
	"fmt"

	"github.com/jetsetilly/test7800/hardware/cpu"
	"github.com/jetsetilly/test7800/hardware/memory/bios"
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
	for _, v := range bios.KnownBIOS {
		if v == md5sum {
			hlp.cartridgeAccepted = true
			break
		}
	}
	hlp.cartridgeAccepted = hlp.cartridgeAccepted && hlp.bypass
	hlp.checksum = false
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
