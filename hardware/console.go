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
	con.TIA = tia.Create(ctx, ui, con.Mem)
	con.RIOT = riot.Create(con.Mem)
	con.MARIA = maria.Create(ctx, ui, con.Mem, con.MC, con.TIA)

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
				if inp.Set {
					con.RIOT.PortWrite(0x00, 0x00, 0xbf)
				} else {
					con.RIOT.PortWrite(0x00, 0x40, 0xbf)
				}
			case ui.StickUp:
				if inp.Set {
					con.RIOT.PortWrite(0x00, 0x00, 0xef)
				} else {
					con.RIOT.PortWrite(0x00, 0x10, 0xef)
				}
			case ui.StickRight:
				if inp.Set {
					con.RIOT.PortWrite(0x00, 0x00, 0x7f)
				} else {
					con.RIOT.PortWrite(0x00, 0x80, 0x7f)
				}
			case ui.StickDown:
				if inp.Set {
					con.RIOT.PortWrite(0x00, 0x00, 0xdf)
				} else {
					con.RIOT.PortWrite(0x00, 0x20, 0xdf)
				}
			case ui.StickButtonA:
				if inp.Set {
					con.TIA.PortWrite(0x0c, 0x00, 0x7f)
				} else {
					con.TIA.PortWrite(0x0c, 0x80, 0x7f)
				}

				// the dual-button stick write to INPT1 has an opposite logic to
				// the write to INPT4/INPT5
				if inp.Set {
					con.TIA.PortWrite(0x09, 0x80, 0x7f)
				} else {
					con.TIA.PortWrite(0x09, 0x00, 0x7f)
				}
			case ui.StickButtonB:
				// the dual-button stick write to INPT0 has an opposite logic to
				// the write to INPT4/INPT5
				if inp.Set {
					con.TIA.PortWrite(0x08, 0x80, 0x7f)
				} else {
					con.TIA.PortWrite(0x08, 0x00, 0x7f)
				}
			case ui.Select:
				if inp.Set {
					con.RIOT.PortWrite(0x02, 0x00, 0xfd)
				} else {
					con.RIOT.PortWrite(0x02, 0x02, 0xfd)
				}
			case ui.Reset:
				if inp.Set {
					con.RIOT.PortWrite(0x02, 0x00, 0xfe)
				} else {
					con.RIOT.PortWrite(0x02, 0x01, 0xfe)
				}
			case ui.P0Pro:
				if inp.Set {
					con.RIOT.PortWrite(0x02, 0x80, 0x7f)
				} else {
					con.RIOT.PortWrite(0x02, 0x00, 0x7f)
				}
			case ui.P1Pro:
				if inp.Set {
					con.RIOT.PortWrite(0x02, 0x40, 0xbf)
				} else {
					con.RIOT.PortWrite(0x02, 0x00, 0xbf)
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
		if con.Mem.IsSlow() {
			mariaCycles = clocks.MariaCycles_for_SlowMemory
		}

		for range mariaCycles {
			var interrupt bool
			con.hlt, con.rdy, interrupt = con.MARIA.Tick()
			interruptNext = interruptNext || interrupt
		}

		con.RIOT.Tick()
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
