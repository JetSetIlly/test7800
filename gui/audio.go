package gui

import (
	"io"

	"github.com/ebitengine/oto/v3"
	"github.com/jetsetilly/test7800/hardware/tia/audio"
)

type sound struct {
	p *oto.Player
	r io.Reader
}

func (s *sound) Read(buf []uint8) (int, error) {
	return s.r.Read(buf)
}

func createAudio(snd chan io.Reader) *sound {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   audio.AverageSampleFreq,
		ChannelCount: 1,
		Format:       oto.FormatSignedInt16LE,
	})

	if err != nil {
		panic(err)
	}

	<-ready

	s := &sound{
		r: <-snd,
	}
	s.p = ctx.NewPlayer(s)
	s.p.Play()

	return s
}
