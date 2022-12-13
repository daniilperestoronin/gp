package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/segmentdisplay"
)

type Pomo struct {
	Count     int
	countDone int
	Tag       string
	Work      time.Duration
	Break     time.Duration
	ctx       context.Context
	cancel    context.CancelFunc
}

func CreatePomo(c int, t string, w, b time.Duration) Pomo {
	return Pomo{
		Count:     c,
		countDone: 1,
		Tag:       t,
		Work:      w,
		Break:     b,
	}
}

func (p *Pomo) Start() {
	t, err := tcell.New()
	if err != nil {
		panic(err)
	}
	defer t.Close()

	p.ctx, p.cancel = context.WithCancel(context.Background())
	clockSD, err := segmentdisplay.New()
	if err != nil {
		panic(err)
	}

	go func() {
		for p.countDone < p.Count+1 {
			p.startTimer(p.Work, cell.ColorRed, clockSD, "Start work")
			p.startTimer(p.Break, cell.ColorGreen, clockSD, "Start rest")

			p.countDone++
		}
	}()

	c, err := container.New(
		t,
		container.Border(linestyle.Light),
		container.BorderTitle(fmt.Sprintf("  Work on: '%s', to stop press Q  ", p.Tag)),

		container.PlaceWidget(clockSD),
	)
	if err != nil {
		panic(err)
	}

	quitter := func(k *terminalapi.Keyboard) {
		if k.Key == 'q' || k.Key == 'Q' {
			p.cancel()
		}
	}

	if err := termdash.Run(p.ctx, t, c, termdash.KeyboardSubscriber(quitter), termdash.RedrawInterval(1*time.Second)); err != nil {
		panic(err)
	}

}

func (p *Pomo) startTimer(w time.Duration, color cell.Color, sd *segmentdisplay.SegmentDisplay, mes string) {
	startTime := time.Now()
	endTime := startTime.Add(w)

	notify(fmt.Sprintf("%s, tag: '%s', round:%d/%d, end time: %s",
		mes, p.Tag, p.countDone, p.Count, endTime.Format("15:04:05")))

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	done := make(chan bool)
	go func() {
		time.Sleep(w)
		done <- true
	}()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			chunks := []*segmentdisplay.TextChunk{
				segmentdisplay.NewChunk(endTime.Sub(t).Round(time.Second).String(), segmentdisplay.WriteCellOpts(cell.FgColor(color))),
			}
			if err := sd.Write(chunks); err != nil {
				panic(err)
			}
		}
	}
}

func notify(message string) {
	err := beeep.Beep(beeep.DefaultFreq, beeep.DefaultDuration)
	if err != nil {
		panic(err)
	}

	err = beeep.Alert("Go Pomo", message, "")
	if err != nil {
		panic(err)
	}
}

func main() {
	wd := flag.String("w", "25m", "Work time duration")
	bd := flag.String("b", "5m", "Break time duration")
	c := flag.Int("c", 5, "Count of rounds")
	t := flag.String("t", "Ordinary task", "Tag")

	w, _ := time.ParseDuration(*wd)
	b, _ := time.ParseDuration(*bd)

	pomo := CreatePomo(*c, *t, w, b)

	pomo.Start()
}
