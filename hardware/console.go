package hardware

import (
	"fmt"

	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware/clocks"
	"github.com/jetsetilly/test7800/hardware/cpu"
	"github.com/jetsetilly/test7800/hardware/maria"
	"github.com/jetsetilly/test7800/hardware/memory"
	"github.com/jetsetilly/test7800/hardware/memory/external"
	"github.com/jetsetilly/test7800/hardware/peripherals"
	"github.com/jetsetilly/test7800/hardware/riot"
	"github.com/jetsetilly/test7800/hardware/tia"
)

type peripheral interface {
	Reset()
	Update(inp gui.Input) error
	Step()
}

type Console struct {
	ctx Context
	g   *gui.GUI

	MC    *cpu.CPU
	Mem   *memory.Memory
	MARIA *maria.Maria
	TIA   *tia.TIA
	RIOT  *riot.RIOT

	panel peripheral

	// the HLT and RDY lines to the CPU is set by MARIA
	hlt bool
	rdy bool

	// counts the number of maria cycles between RIOT/TIA ticks, regardless of current CPU speed
	cycleRegulator int

	// frame limiter
	limit *limiter
}

type Context interface {
	memory.Context
	tia.Context
	maria.Context
	Rand8Bit() uint8
	Rand16Bit() uint16
	UseAudio() bool
}

func Create(ctx Context, g *gui.GUI) *Console {
	spec := ctx.Spec()

	con := &Console{
		ctx:   ctx,
		g:     g,
		limit: newLimiter(spec, ctx.UseAudio()),
	}

	// create and attach console components
	var addChips memory.AddChips
	con.Mem, addChips = memory.Create(ctx)

	con.MC = cpu.Create(con.Mem)
	con.RIOT = riot.Create()
	con.TIA = tia.Create(ctx, g, con.RIOT, con.limit)
	con.MARIA = maria.Create(ctx, g, con.Mem, con.MC, con.limit)

	addChips(con.MARIA, con.TIA, con.RIOT)

	con.panel = peripherals.NewPanel(con.RIOT)

	return con
}

// if biosCheck is nil or if it returns false then the BIOS routines are bypassed
func (con *Console) Reset(random bool, biosCheck func() bool) error {
	var rnd cpu.Random
	if random {
		rnd = con.ctx
	}

	con.Mem.Reset(random)
	con.RIOT.Reset()
	con.TIA.Reset()
	con.MARIA.Reset()

	// reset CPU after memory reset so that we get the correct reset address (the BIOS might be locked)
	con.MC.Reset(rnd)

	if biosCheck == nil || !biosCheck() {
		// writing to the INPTCTRL twice to make sure the halt line has been enabled
		con.Mem.INPTCTRL.Write(0x01, 0x07)
		con.Mem.INPTCTRL.Write(0x01, 0x07)

		// explicitely set 6507 program-counter to reset address when the BIOS is disabled
		err := con.MC.LoadPCIndirect(cpu.Reset)
		if err != nil {
			return err
		}
	}

	return nil
}

func (con *Console) Insert(c external.CartridgeInsertor) error {
	err := con.Mem.External.Insert(c)
	if err != nil {
		return err
	}
	err = con.TIA.Insert(c, con.Mem.External.Chips)
	if err != nil {
		return err
	}
	return nil
}

func (con *Console) Step() error {
	// handle input (left stick only)
	var drained bool
	for !drained {
		select {
		default:
			drained = true
		case inp := <-con.g.UserInput:
			if inp.Port == gui.Panel {
				con.panel.Update(inp)
			}
			if inp.Port == gui.Player0 {
				switch inp.Action {
				case gui.StickLeft:
					if inp.Data.(bool) {
						// unset the opposite direction first (applies to all
						// other directions below)
						con.RIOT.PortWrite(0x00, 0x80, 0x7f)
						con.RIOT.PortWrite(0x00, 0x00, 0xbf)
					} else {
						con.RIOT.PortWrite(0x00, 0x40, 0xbf)
					}
				case gui.StickUp:
					if inp.Data.(bool) {
						con.RIOT.PortWrite(0x00, 0x20, 0xdf)
						con.RIOT.PortWrite(0x00, 0x00, 0xef)
					} else {
						con.RIOT.PortWrite(0x00, 0x10, 0xef)
					}
				case gui.StickRight:
					if inp.Data.(bool) {
						con.RIOT.PortWrite(0x00, 0x40, 0xbf)
						con.RIOT.PortWrite(0x00, 0x00, 0x7f)
					} else {
						con.RIOT.PortWrite(0x00, 0x80, 0x7f)
					}
				case gui.StickDown:
					if inp.Data.(bool) {
						con.RIOT.PortWrite(0x00, 0x10, 0xef)
						con.RIOT.PortWrite(0x00, 0x00, 0xdf)
					} else {
						con.RIOT.PortWrite(0x00, 0x20, 0xdf)
					}
				case gui.StickButtonA:
					// https://forums.atariage.com/topic/127162-question-about-joysticks-and-how-they-are-read/#findComment-1537159
					b, err := con.RIOT.Read(0x02)
					if err != nil {
						return fmt.Errorf("stick button a: %w", err)
					}
					if b&0x04 == 0x04 {
						if inp.Data.(bool) {
							con.TIA.PortWrite(0x0c, 0x00, 0x7f)
						} else {
							con.TIA.PortWrite(0x0c, 0x80, 0x7f)
						}
					} else {
						// the two-button stick write to INPT0/INPT1 has an opposite logic to
						// the write to INPT4/INPT5
						if inp.Data.(bool) {
							con.TIA.PortWrite(0x09, 0x80, 0x7f)
						} else {
							con.TIA.PortWrite(0x09, 0x00, 0x7f)
						}
					}
				case gui.StickButtonB:
					// https://forums.atariage.com/topic/127162-question-about-joysticks-and-how-they-are-read/#findComment-1537159
					b, err := con.RIOT.Read(0x02)
					if err != nil {
						return fmt.Errorf("stick button b: %w", err)
					}
					if b&0x04 == 0x04 {
						if inp.Data.(bool) {
							con.TIA.PortWrite(0x0c, 0x00, 0x7f)
						} else {
							con.TIA.PortWrite(0x0c, 0x80, 0x7f)
						}
					} else {
						// the two-button stick write to INPT0/INPT1 has an opposite logic to
						// the write to INPT4/INPT5
						if inp.Data.(bool) {
							con.TIA.PortWrite(0x08, 0x80, 0x7f)
						} else {
							con.TIA.PortWrite(0x08, 0x00, 0x7f)
						}
					}
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
	var tick func() error
	tick = func() error {
		mariaCycles := clocks.MariaCycles
		if con.Mem.IsSlowAddressBus() {
			mariaCycles = clocks.MariaCycles_for_SlowMemory
		}

		for i := range mariaCycles {
			var interrupt bool
			con.hlt, con.rdy, interrupt = con.MARIA.Tick(i == mariaCycles-1)
			interruptNext = interruptNext || interrupt

			con.cycleRegulator++
			if con.cycleRegulator > 3 {
				con.TIA.Tick()
				con.RIOT.Tick()
				con.cycleRegulator = 0
			}
		}

		// consume DMA cycles (but not WSYNC cycles)
		for con.hlt && con.Mem.INPTCTRL.HaltEnabled() {
			err := tick()
			if err != nil {
				return err
			}
		}

		return nil
	}

	// swallow all DMA activity. the CPU will be halted during this time so. INPTCTRL only allows
	// the HALT line to be raised after an initial phase. the HaltEnabled() function tells us the
	// state of that condition. WSYNC also causes HALT to be enabled
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
		case <-con.g.UserInput:
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
