package hardware

import (
	"github.com/jetsetilly/test7800/hardware/clocks"
	"github.com/jetsetilly/test7800/hardware/cpu"
	"github.com/jetsetilly/test7800/hardware/maria"
	"github.com/jetsetilly/test7800/hardware/memory"
	"github.com/jetsetilly/test7800/hardware/riot"
	"github.com/jetsetilly/test7800/hardware/tia"
	"github.com/jetsetilly/test7800/ui"
)

type Console struct {
	ctx Context
	ui  *ui.UI

	MC    *cpu.CPU
	Mem   *memory.Memory
	MARIA *maria.Maria
	TIA   *tia.TIA
	RIOT  *riot.RIOT

	// the HLT and RDY lines to the CPU is set by MARIA
	hlt bool
	rdy bool
}

type Context interface {
	cpu.Context
	memory.Context
	tia.Context
	maria.Context
	Rand8Bit() uint8
	Rand16Bit() uint16
}

func Create(ctx Context, ui *ui.UI) Console {
	con := Console{
		ctx: ctx,
		ui:  ui,
	}

	var addChips memory.AddChips
	con.Mem, addChips = memory.Create(ctx)

	con.MC = cpu.NewCPU(ctx, con.Mem)
	con.MARIA = maria.Create(ctx, ui, con.Mem, con.MC)
	con.TIA = tia.Create(ctx, ui, con.Mem)
	con.RIOT = riot.Create(con.Mem)

	addChips(con.MARIA, con.TIA, con.RIOT)

	con.Reset(true)
	return con
}

func (con *Console) Reset(random bool) error {
	con.MC.Reset()
	if random {
		con.MC.PC.Load(con.ctx.Rand16Bit())
		con.MC.A.Load(con.ctx.Rand8Bit())
		con.MC.X.Load(con.ctx.Rand8Bit())
		con.MC.Y.Load(con.ctx.Rand8Bit())
	}
	con.Mem.Reset(random)
	con.MARIA.Reset()

	return con.MC.LoadPCIndirect(cpu.Reset)
}

func (con *Console) Step() error {
	// handle input (left stick only)
	var drained bool
	for !drained {
		select {
		default:
			drained = true
		case inp := <-con.ui.UserInput:
			switch inp.Action {
			case ui.StickLeft:
				r, _ := con.RIOT.Read(0x00)
				if inp.Release {
					con.RIOT.Poke(0x00, r|0x40)
				} else {
					con.RIOT.Poke(0x00, r&0xbf)
				}
				r, _ = con.RIOT.Read(0x00)
			case ui.StickUp:
				r, _ := con.RIOT.Read(0x00)
				if inp.Release {
					con.RIOT.Poke(0x00, r|0x10)
				} else {
					con.RIOT.Poke(0x00, r&0xef)
				}
			case ui.StickRight:
				r, _ := con.RIOT.Read(0x00)
				if inp.Release {
					con.RIOT.Poke(0x00, r|0x80)
				} else {
					con.RIOT.Poke(0x00, r&0x7f)
				}
			case ui.StickDown:
				r, _ := con.RIOT.Read(0x00)
				if inp.Release {
					con.RIOT.Poke(0x00, r|0x20)
				} else {
					con.RIOT.Poke(0x00, r&0xdf)
				}
			case ui.StickButtonA:
				r, _ := con.TIA.Read(0x0c)
				if inp.Release {
					con.TIA.Poke(0x0c, r|0x80)
				} else {
					con.TIA.Poke(0x0c, r&0x7f)
				}

				// the dual-button stick write to INPT1 has an opposite logic to
				// the write to INPT4/INPT5
				r, _ = con.TIA.Read(0x09)
				if inp.Release {
					con.TIA.Poke(0x09, r&0x7f)
				} else {
					con.TIA.Poke(0x09, r|0x80)
				}
			case ui.StickButtonB:
				// the dual-button stick write to INPT0 has an opposite logic to
				// the write to INPT4/INPT5
				r, _ := con.TIA.Read(0x08)
				if inp.Release {
					con.TIA.Poke(0x08, r&0x7f)
				} else {
					con.TIA.Poke(0x08, r|0x80)
				}
			case ui.Select:
				r, _ := con.RIOT.Read(0x02)
				if inp.Release {
					con.RIOT.Poke(0x02, r|0x02)
				} else {
					con.RIOT.Poke(0x02, r&0xfd)
				}
			case ui.Reset:
				r, _ := con.RIOT.Read(0x02)
				if inp.Release {
					con.RIOT.Poke(0x02, r|0x01)
				} else {
					con.RIOT.Poke(0x02, r&0xfe)
				}
			case ui.P0Pro:
				r, _ := con.RIOT.Read(0x02)
				if !inp.Release {
					con.RIOT.Poke(0x02, r^0x80)
				}
			case ui.P1Pro:
				r, _ := con.RIOT.Read(0x02)
				if !inp.Release {
					con.RIOT.Poke(0x02, r^0x40)
				}
			}
		}
	}

	// interrupts are atomic, meaning that the interrupt occurs between
	// instruction boundaries and never during an instruction
	var interruptNext bool
	defer func() {
		if interruptNext {
			_ = con.MC.Interrupt(true)
		}
	}()

	// this function is called once per CPU cycle. MARIA runs faster than
	// the CPU and so there are multiple ticks of the MARIA per CPU cycle
	//
	// if the TIA bus is active then the CPU runs at a slower clock
	tick := func() error {
		mariaCycles := clocks.MariaCycles
		if con.Mem.IsTIA() {
			mariaCycles = clocks.MariaCycles_for_TIA
		}

		for range mariaCycles {
			var interrupt bool
			con.hlt, con.rdy, interrupt = con.MARIA.Tick()
			interruptNext = interruptNext || interrupt
		}

		con.TIA.Tick()

		return nil
	}

	// swallow all DMA activity. the CPU will be halted during this time so.
	// INPTCTRL only allows the HALT line to be raised after an initial
	// phase. the HaltEnabled() function tells us the state of that condition
	//
	// WSYNC also causes HALT to be enabled
	for (con.hlt && con.Mem.INPTCTRL.HaltEnabled()) || !con.rdy {
		err := tick()
		if err != nil {
			return err
		}
	}

	return con.MC.ExecuteInstruction(tick)
}

func (con *Console) Run(hook func() error) error {
	// drain input channel
	var drained bool
	for !drained {
		select {
		case <-con.ui.UserInput:
		default:
			drained = true
		}
	}

	for {
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
	if con.Mem.LastWrite == nil {
		return ""
	}
	var s string
	l, ok := con.Mem.LastWrite.(lastArea)
	if ok {
		s = l.Status()
	}
	con.Mem.LastWrite = nil
	return s
}
