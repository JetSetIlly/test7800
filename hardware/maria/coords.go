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

func (c *coords) Reset() {
	c.Frame = 0
	c.Scanline = 0
	c.Clk = 0
}
