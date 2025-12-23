package hardware

import (
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
	IsAnalogue() bool
	Reset()
	Unplug()
	Update(inp gui.Input) error
	Tick()
}

type Console struct {
	ctx Context
	g   *gui.GUI

	MC    *cpu.CPU
	Mem   *memory.Memory
	MARIA *maria.Maria
	TIA   *tia.TIA
	RIOT  *riot.RIOT

	panel   peripheral
	players [2]peripheral

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
	con.TIA = tia.Create(ctx, g, con.limit)
	con.MARIA = maria.Create(ctx, g, con.Mem, con.MC, con.limit)

	addChips(con.MARIA, con.TIA, con.RIOT)

	con.panel = peripherals.NewPanel(con.RIOT)
	con.players[0] = peripherals.NewStick(con.RIOT, con.TIA, false, true)
	con.players[1] = peripherals.NewStick(con.RIOT, con.TIA, true, true)
	con.panel.Reset()
	con.players[0].Reset()
	con.players[1].Reset()

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

	con.panel.Reset()
	con.players[0].Reset()
	con.players[1].Reset()

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
	err = con.TIA.Insert(con.Mem.External.Chips)
	if err != nil {
		return err
	}

	switch c.Controller {
	case "7800_joystick":
		if _, ok := con.players[0].(*peripherals.Stick); !ok {
			con.players[0].Unplug()
			con.players[0] = peripherals.NewStick(con.RIOT, con.TIA, false, true)
			con.players[0].Reset()
		}
		if _, ok := con.players[1].(*peripherals.Stick); !ok {
			con.players[1].Unplug()
			con.players[1] = peripherals.NewStick(con.RIOT, con.TIA, false, true)
			con.players[1].Reset()
		}
	case "paddle":
		if _, ok := con.players[0].(*peripherals.Paddles); !ok {
			con.players[0].Unplug()
			con.players[0] = peripherals.NewPaddles(con.RIOT, con.TIA, false)
			con.players[0].Reset()
		}
		if _, ok := con.players[1].(*peripherals.Paddles); !ok {
			con.players[1].Unplug()
			con.players[1] = peripherals.NewPaddles(con.RIOT, con.TIA, true)
			con.players[1].Reset()
		}
	case "trakball":
		if _, ok := con.players[0].(*peripherals.Trakball); !ok {
			con.players[0].Unplug()
			con.players[0] = peripherals.NewTrakball(con.RIOT, con.TIA, con.Mem, false)
			con.players[0].Reset()
		}
		if _, ok := con.players[1].(*peripherals.Trakball); !ok {
			con.players[1].Unplug()
			con.players[1] = peripherals.NewTrakball(con.RIOT, con.TIA, con.Mem, true)
			con.players[1].Reset()
		}
	case "2600_joystick":
		if _, ok := con.players[0].(*peripherals.Stick); !ok {
			con.players[0].Unplug()
			con.players[0] = peripherals.NewStick(con.RIOT, con.TIA, false, false)
			con.players[0].Reset()
		}
		if _, ok := con.players[1].(*peripherals.Stick); !ok {
			con.players[1].Unplug()
			con.players[1] = peripherals.NewStick(con.RIOT, con.TIA, false, true)
			con.players[1].Reset()
		}
	case "snes2atari":
		if _, ok := con.players[0].(*peripherals.Stick); !ok {
			con.players[0].Unplug()
			con.players[0] = peripherals.NewStick(con.RIOT, con.TIA, false, true)
			con.players[0].Reset()
		}
		if _, ok := con.players[1].(*peripherals.Stick); !ok {
			con.players[1].Unplug()
			con.players[1] = peripherals.NewStick(con.RIOT, con.TIA, false, true)
			con.players[1].Reset()
		}
	}
	return nil
}

func (con *Console) Step() error {
	con.handleInput()

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
				con.panel.Tick()
				con.players[0].Tick()
				con.players[1].Tick()
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
