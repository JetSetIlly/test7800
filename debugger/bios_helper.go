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

	// set to false to ignore checksum failure in supported BIOS'
	skipChecksum bool

	// a copy of the BIOS info
	bios bios.Info

	// checksum has passed. there is no need to continue checking for a fail condition
	cartridgePassed bool
}

func (hlp *biosHelper) reset(md5sum string) {
	hlp.bios = bios.Info{}

	for _, v := range bios.Supported {
		if v.MD5 == md5sum {
			hlp.bios = v
			break
		}
	}

	hlp.cartridgePassed = hlp.bios.Name == "" || hlp.bios.MD5 == "" || hlp.bypass || !hlp.bios.ChecksSignature
}

func (hlp *biosHelper) cartridgePassCheck(mc *cpu.CPU) error {
	if hlp.cartridgePassed {
		return nil
	}
	if mc.PC.Address() == hlp.bios.SignatureFail {
		if !hlp.skipChecksum {
			return fmt.Errorf("checksum fail. ROM not signed for BIOS")
		}
		mc.PC.Load(hlp.bios.SignaturePass)
	}
	if mc.PC.Address() == hlp.bios.SignaturePass {
		hlp.cartridgePassed = true
	}
	return nil
}
