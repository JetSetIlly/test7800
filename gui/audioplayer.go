package gui

import (
	"io"

	"github.com/ebitengine/oto/v3"
	"github.com/jetsetilly/test7800/hardware/tia/audio"
	"github.com/jetsetilly/test7800/ui"
)

type audioPlayer struct {
	p *oto.Player
	r io.Reader
}

func (s *audioPlayer) Read(buf []uint8) (int, error) {
	return s.r.Read(buf)
}

func createAudioPlayer(ui *ui.UI) *audioPlayer {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   audio.AverageSampleFreq,
		ChannelCount: 1,
		Format:       oto.FormatSignedInt16LE,
	})

	if err != nil {
		panic(err)
	}

	<-ready

	s := &audioPlayer{
		r: <-ui.RegisterAudio,
	}
	s.p = ctx.NewPlayer(s)
	s.p.Play()

	return s
}
