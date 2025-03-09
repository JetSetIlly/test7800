package hardware

import (
	"math/rand/v2"

	_ "embed"

	"github.com/jetsetilly/test7800/hardware/cpu"
	"github.com/jetsetilly/test7800/hardware/ram"
)

type Console struct {
	MC  *cpu.CPU
	Mem *memory
}

func Create() Console {
	var con Console
	con.Mem = &memory{
		RAM7800: ram.Create("ram7800", 0x1000),
		RAMRIOT: ram.Create("ramRIOT", 0x0080),
	}
	con.MC = cpu.NewCPU(con.Mem)
	con.Reset(true)
	return con
}

func (con *Console) Reset(random bool) error {
	con.MC.Reset()
	if random {
		con.MC.PC.Load(uint16(rand.IntN(65535)))
		con.MC.A.Load(uint8(rand.IntN(255)))
		con.MC.X.Load(uint8(rand.IntN(255)))
		con.MC.Y.Load(uint8(rand.IntN(255)))
	}
	con.Mem.RAM7800.Reset(random)
	con.Mem.RAMRIOT.Reset(random)

	return con.MC.LoadPCIndirect(cpu.Reset)
}

func (con *Console) Step() error {
	cycle := func() error {
		return nil
	}
	return con.MC.ExecuteInstruction(cycle)
}

func (con *Console) Run(stop chan bool, hook func() error) error {
	defer func() {
		con.Mem.last = nil
	}()

	for {
		select {
		case <-stop:
			return nil
		default:
		}

		err := con.Step()
		if err != nil {
			return err
		}

		err = hook()
		if err != nil {
			return err
		}
	}
}

func (con *Console) LastMemoryAccess() string {
	if con.Mem.last == nil {
		return ""
	}
	s := con.Mem.last.Status()
	con.Mem.last = nil
	return s
}
