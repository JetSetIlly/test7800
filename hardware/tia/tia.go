package tia

import "github.com/jetsetilly/test7800/debugger/dbg"

type TIA struct {
	ctx *dbg.Context
	mem Memory
}

type Memory interface {
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
}

func Create(ctx *dbg.Context, mem Memory) *TIA {
	return &TIA{
		ctx: ctx,
		mem: mem,
	}
}

func (tia *TIA) Label() string {
	return "TIA"
}

func (tia *TIA) Status() string {
	return tia.Label()
}

func (tia *TIA) Read(address uint16) (uint8, error) {
	return 0, nil
}

func (tia *TIA) Write(address uint16, data uint8) error {
	return nil
}
