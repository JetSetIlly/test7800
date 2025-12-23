package peripherals

import (
	"github.com/jetsetilly/test7800/hardware/riot"
	"github.com/jetsetilly/test7800/hardware/tia"
)

type RIOT interface {
	PortWrite(reg riot.Register, data uint8, mask uint8) error
	PortRead(reg riot.Register) (uint8, error)
}

type TIA interface {
	PortWrite(reg tia.Register, data uint8, mask uint8) error
}

type PaddlesTIA interface {
	TIA
	PaddlesGrounded() bool
}

type Memory interface {
	LastReadIsRIOT() bool
}
