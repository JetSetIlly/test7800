package debugger

import (
	"errors"

	"github.com/jetsetilly/test7800/gui"
	"github.com/jetsetilly/test7800/hardware"
	"github.com/jetsetilly/test7800/hardware/memory/external"
)

// preview is a console context without any video/audio output used for gathering information about
// a ROM that can't be determined statically
type preview struct {
	ctx     context
	console *hardware.Console
	g       *gui.ChannelsDebugger
}

func newPreview(ctx context) (*preview, error) {
	p := &preview{
		g: gui.NewChannels().Debugger(),
		ctx: context{
			requestedSpec: ctx.requestedSpec,
			overscan:      ctx.overscan,
		},
	}
	p.ctx.Reset()
	p.console = hardware.Create(&p.ctx, p.g)
	return p, nil
}

func (p *preview) run(loader external.CartridgeInsertor) error {
	err := p.console.Insert(loader)
	if err != nil {
		return err
	}

	err = p.console.Reset(true, nil)
	if err != nil {
		return err
	}

	var endRunErr = errors.New("end run")

	err = p.console.Run(func() error {
		if p.console.MARIA.Coords.Frame >= 30 {
			return endRunErr
		}
		return nil
	})

	if !errors.Is(err, endRunErr) {
		return err
	}

	return nil
}

func (p *preview) sync(console *hardware.Console) {
	t, b := p.console.MARIA.GetOverscan()
	console.MARIA.SetOverscan(t, b)
}
