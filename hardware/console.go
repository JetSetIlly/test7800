package hardware

import (
	"image"
	"math/rand/v2"

	"github.com/jetsetilly/test7800/hardware/clocks"
	"github.com/jetsetilly/test7800/hardware/cpu"
	"github.com/jetsetilly/test7800/hardware/maria"
	"github.com/jetsetilly/test7800/hardware/memory"
	"github.com/jetsetilly/test7800/hardware/riot"
	"github.com/jetsetilly/test7800/hardware/tia"
)

type Console struct {
	MC    *cpu.CPU
	Mem   *memory.Memory
	MARIA *maria.Maria
	TIA   *tia.TIA
	RIOT  *riot.RIOT

	// the HLT line to the CPU is set by MARIA
	halt bool
}

func Create(rendering chan *image.RGBA) Console {
	con := Console{
		TIA:  &tia.TIA{},
		RIOT: &riot.RIOT{},
	}

	var addChips memory.AddChips
	con.Mem, addChips = memory.Create()

	con.MC = cpu.NewCPU(con.Mem)
	con.MARIA = maria.Create(con.Mem, con.Mem.BIOS.Spec(), rendering)
	addChips(con.MARIA, con.TIA, con.RIOT)

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
	con.Mem.Reset(random)

	return con.MC.LoadPCIndirect(cpu.Reset)
}

func (con *Console) Step() error {
	var nmiNextInstruction bool

	defer func() {
		if nmiNextInstruction {
			_ = con.MC.Interrupt(true)
		}
	}()

	tick := func() error {
		// this function is called once per CPU cycle. MARIA runs faster than
		// the CPU and so there are multiple ticks of the MARIA per CPU cycle
		//
		// if the TIA bus is active then the CPU runs at a slower clock
		//
		// TODO: handle slowing down of CPU
		for range clocks.MariaCycles {
			var nmi bool
			con.halt, nmi = con.MARIA.Tick()
			nmiNextInstruction = nmiNextInstruction || nmi
		}
		return nil
	}

	if con.halt && con.Mem.INPTCTRL.HaltEnabled() {
		// swallow all DMA activity. the CPU will be halted during this time so.
		// INPTCTRL only allows the HALT line to be raised after an initial
		// phase. the HaltEnabled() function tells us the state of that condition
		for con.halt && con.Mem.INPTCTRL.HaltEnabled() {
			err := tick()
			if err != nil {
				return err
			}
		}

		// deliberately fall through to ExecuteInstruction()
	}

	return con.MC.ExecuteInstruction(tick)
}

func (con *Console) Run(stop chan bool, hook func() error) error {
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

type lastArea interface {
	Status() string
}

// LastAreaStatus returns the status of the last memory area to be written to
// (if the memory area provides the Status() function).
//
// Once this function has been executed, the last memory area information is gone
// and it will return nothing until the next memory write.
func (con *Console) LastAreaStatus() string {
	if con.Mem.Last == nil {
		return ""
	}
	var s string
	l, ok := con.Mem.Last.(lastArea)
	if ok {
		s = l.Status()
	}
	con.Mem.Last = nil
	return s
}
