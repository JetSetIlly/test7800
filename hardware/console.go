package hardware

import (
	"image"

	"github.com/jetsetilly/test7800/hardware/clocks"
	"github.com/jetsetilly/test7800/hardware/cpu"
	"github.com/jetsetilly/test7800/hardware/maria"
	"github.com/jetsetilly/test7800/hardware/memory"
	"github.com/jetsetilly/test7800/hardware/riot"
	"github.com/jetsetilly/test7800/hardware/tia"
	"github.com/jetsetilly/test7800/io"
)

type Console struct {
	ctx Context
	inp chan io.Input

	MC    *cpu.CPU
	Mem   *memory.Memory
	MARIA *maria.Maria
	TIA   *tia.TIA
	RIOT  *riot.RIOT

	// the HLT line to the CPU is set by MARIA
	halt bool
}

type Context interface {
	cpu.Context
	memory.Context
	Rand8Bit() uint8
	Rand16Bit() uint16
}

func Create(ctx Context, rendering chan *image.RGBA, inp chan io.Input) Console {
	con := Console{
		ctx: ctx,
		inp: inp,
	}

	var addChips memory.AddChips
	con.Mem, addChips = memory.Create(ctx)

	con.MC = cpu.NewCPU(ctx, con.Mem)
	con.MARIA = maria.Create(ctx, con.Mem, con.Mem.BIOS.Spec(), rendering)
	con.TIA = tia.Create(con.Mem)
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
		case inp := <-con.inp:
			switch inp.Action {
			case io.StickButtonA:
				if inp.Release {
					con.TIA.Write(0x0c, 0x80)
				} else {
					con.TIA.Write(0x0c, 0x00)
				}
			case io.StickLeft:
				r, _ := con.RIOT.Read(0x00)
				if inp.Release {
					con.RIOT.Write(0x00, r&0xbf|0x40)
				} else {
					con.RIOT.Write(0x00, r&0xbf)
				}
				r, _ = con.RIOT.Read(0x00)
			case io.StickUp:
				r, _ := con.RIOT.Read(0x00)
				if inp.Release {
					con.RIOT.Write(0x00, r&0xef|0x10)
				} else {
					con.RIOT.Write(0x00, r&0xef)
				}
			case io.StickRight:
				r, _ := con.RIOT.Read(0x00)
				if inp.Release {
					con.RIOT.Write(0x00, r&0x7f|0x80)
				} else {
					con.RIOT.Write(0x00, r&0x7f)
				}
			case io.StickDown:
				r, _ := con.RIOT.Read(0x00)
				if inp.Release {
					con.RIOT.Write(0x00, r&0xdf|0x20)
				} else {
					con.RIOT.Write(0x00, r&0xdf)
				}
			}
		}
	}

	var interruptNext bool

	defer func() {
		if interruptNext {
			_ = con.MC.Interrupt(true)
		}
	}()

	tick := func() error {
		// the CPU slows down when TIA memory has been accessed
		mariaCycles := clocks.MariaCycles
		if con.Mem.IsTIA() {
			mariaCycles = clocks.MariaCycles_for_TIA
		}

		// this function is called once per CPU cycle. MARIA runs faster than
		// the CPU and so there are multiple ticks of the MARIA per CPU cycle
		//
		// if the TIA bus is active then the CPU runs at a slower clock
		for range mariaCycles {
			var interrupt bool
			con.halt, interrupt = con.MARIA.Tick()
			interruptNext = interruptNext || interrupt
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

func (con *Console) Run(hook func() error) error {
	// drain input channel
	var drained bool
	for !drained {
		select {
		case <-con.inp:
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
