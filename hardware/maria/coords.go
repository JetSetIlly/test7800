package maria

import (
	"fmt"
)

type coords struct {
	Frame    int
	Scanline int
	Clk      int
}

func (c *coords) String() string {
	return fmt.Sprintf("frame: %d, scanline: %d, clk: %d", c.Frame, c.Scanline, c.Clk)
}

func (c *coords) ShortString() string {
	return fmt.Sprintf("%d/%03d/%03d", c.Frame, c.Scanline, c.Clk)
}

type random interface {
	RandN(int) int
}

func (c *coords) reset(rnd random) {
	c.Frame = 0
	c.Scanline = 0
	c.Clk = 0

	// it's not certain there is any randomness in the initial state of maria coordinates. however,
	// adding a small variation in the starting clock reflects observations seen in hardware for
	// some ROMs. for example, the "tiamariasync.bas.bin" ROM shows some variation on hardware
	//
	// there are surely other areas where randomness can be added with the same result (the starting
	// scanline for example) but a slight variation in clk seems more likely and less obtrusive
	if rnd != nil {
		c.Clk = rnd.RandN(3)
	}
}
