package hardware

import (
	"fmt"

	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware/clocks"
	"github.com/jetsetilly/test7800/hardware/cpu"
	"github.com/jetsetilly/test7800/hardware/maria"
	"github.com/jetsetilly/test7800/hardware/memory"
	"github.com/jetsetilly/test7800/hardware/memory/external"
	"github.com/jetsetilly/test7800/hardware/riot"
	"github.com/jetsetilly/test7800/hardware/tia"
	"github.com/jetsetilly/test7800/hardware/tia/audio"
)

type Console struct {
	ctx Context
	g   *gui.GUI

	MC    *cpu.CPU
	Mem   *memory.Memory
	MARIA *maria.Maria
	TIA   *tia.TIA
	RIOT  *riot.RIOT

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
		limit: newLimiter(spec),
	}

	// create and attach console components
	var addChips memory.AddChips
	con.Mem, addChips = memory.Create(ctx)

	con.MC = cpu.Create(con.Mem)
	con.RIOT = riot.Create()
	con.TIA = tia.Create(ctx, g, con.RIOT, con.limit)
	con.MARIA = maria.Create(ctx, g, con.Mem, con.MC, con.limit)

	addChips(con.MARIA, con.TIA, con.RIOT)

	// notify UI of audio requirements
	select {
	case g.AudioSetup <- gui.AudioSetup{
		Freq: spec.HorizScan * audio.SamplesPerScanline,
		Read: con.TIA.AudioBuffer(),
		Mute: !ctx.UseAudio(),
	}:
	default:
	}

	return con
}

func (con *Console) Reset(random bool) error {
	var rnd cpu.Random
	if random {
		rnd = con.ctx
	}
	con.MC.Reset(rnd)
	con.Mem.Reset(random)
	con.MARIA.Reset()

	return nil
}

func (con *Console) Insert(c external.CartridgeInsertor) error {
	err := con.RIOT.Insert(c)
	if err != nil {
		return err
	}
	return con.Mem.External.Insert(c)
}

func (con *Console) Step() error {
	// handle input (left stick only)
	var drained bool
	for !drained {
		select {
		default:
			drained = true
		case inp := <-con.g.UserInput:
			switch inp.Action {
			case gui.StickLeft:
				if inp.Set {
					con.RIOT.PortWrite(0x00, 0x00, 0xbf)
				} else {
					con.RIOT.PortWrite(0x00, 0x40, 0xbf)
				}
			case gui.StickUp:
				if inp.Set {
					con.RIOT.PortWrite(0x00, 0x00, 0xef)
				} else {
					con.RIOT.PortWrite(0x00, 0x10, 0xef)
				}
			case gui.StickRight:
				if inp.Set {
					con.RIOT.PortWrite(0x00, 0x00, 0x7f)
				} else {
					con.RIOT.PortWrite(0x00, 0x80, 0x7f)
				}
			case gui.StickDown:
				if inp.Set {
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
					if inp.Set {
						con.TIA.PortWrite(0x0c, 0x00, 0x7f)
					} else {
						con.TIA.PortWrite(0x0c, 0x80, 0x7f)
					}
				} else {
					// the two-button stick write to INPT0/INPT1 has an opposite logic to
					// the write to INPT4/INPT5
					if inp.Set {
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
					if inp.Set {
						con.TIA.PortWrite(0x0c, 0x00, 0x7f)
					} else {
						con.TIA.PortWrite(0x0c, 0x80, 0x7f)
					}
				} else {
					// the two-button stick write to INPT0/INPT1 has an opposite logic to
					// the write to INPT4/INPT5
					if inp.Set {
						con.TIA.PortWrite(0x08, 0x80, 0x7f)
					} else {
						con.TIA.PortWrite(0x08, 0x00, 0x7f)
					}
				}
			case gui.Select:
				if inp.Set {
					con.RIOT.PortWrite(0x02, 0x00, 0xfd)
				} else {
					con.RIOT.PortWrite(0x02, 0x02, 0xfd)
				}
			case gui.Start:
				if inp.Set {
					con.RIOT.PortWrite(0x02, 0x00, 0xfe)
				} else {
					con.RIOT.PortWrite(0x02, 0x01, 0xfe)
				}
			case gui.Pause:
				if inp.Set {
					con.RIOT.PortWrite(0x02, 0x00, 0xf7)
				} else {
					con.RIOT.PortWrite(0x02, 0x08, 0xf7)
				}
			case gui.P0Pro:
				if inp.Set {
					con.RIOT.PortWrite(0x02, 0x80, 0x7f)
				} else {
					con.RIOT.PortWrite(0x02, 0x00, 0x7f)
				}
			case gui.P1Pro:
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
		if con.Mem.IsSlowAddressBus() {
			mariaCycles = clocks.MariaCycles_for_SlowMemory
		}

		for range mariaCycles {
			var interrupt bool
			con.hlt, con.rdy, interrupt = con.MARIA.Tick()
			interruptNext = interruptNext || interrupt

			con.cycleRegulator++
			if con.cycleRegulator > 5 {
				con.RIOT.Tick()
				con.TIA.Tick()
				con.cycleRegulator = 0
			}
		}

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
