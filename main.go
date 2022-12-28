package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/segmentdisplay"
)

type TimerType int

const (
	Work TimerType = iota
	Break
)

type Pomo struct {
	Round        int
	currentRound int
	Tag          string
	Work         time.Duration
	Break        time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
	timer        chan string
	timerType    chan TimerType
}

func CreatePomo(r int, t string, w, b time.Duration) Pomo {
	return Pomo{
		Round:        r,
		currentRound: 1,
		Tag:          t,
		Work:         w,
		Break:        b,
		timer:        make(chan string),
		timerType:    make(chan TimerType),
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

	go systray.Run(p.onReady, p.onExit)
	go func() {
		for p.currentRound < p.Round+1 {
			p.startTimer(p.Work, cell.ColorRed, clockSD, Work)
			p.startTimer(p.Break, cell.ColorGreen, clockSD, Break)

			p.currentRound++
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

func (p *Pomo) startTimer(w time.Duration, color cell.Color, sd *segmentdisplay.SegmentDisplay, tt TimerType) {
	p.timerType <- tt

	startTime := time.Now()
	endTime := startTime.Add(w)

	var mes string
	if tt == Work {
		mes = "Start work"
	} else {
		mes = "Start break"
	}

	notify(fmt.Sprintf("%s, tag: '%s', round:%d/%d, end time: %s",
		mes, p.Tag, p.currentRound, p.Round, endTime.Format("15:04:05")))

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
			ct := endTime.Sub(t).Round(time.Second).String()
			p.timer <- ct
			chunks := []*segmentdisplay.TextChunk{
				segmentdisplay.NewChunk(ct, segmentdisplay.WriteCellOpts(cell.FgColor(color))),
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

	err = beeep.Alert("GPom", message, "")
	if err != nil {
		panic(err)
	}
}

func main() {
	wd := flag.String("w", "25m", "Work time duration")
	bd := flag.String("b", "5m", "Break time duration")
	r := flag.Int("r", 5, "Count of rounds")
	t := flag.String("t", "Ordinary task", "Tag")

	w, _ := time.ParseDuration(*wd)
	b, _ := time.ParseDuration(*bd)

	pomo := CreatePomo(*r, *t, w, b)

	pomo.Start()
}

func (p *Pomo) onReady() {
	gpomIcon, err := ioutil.ReadFile("assets/gpom.ico")
	if err != nil {
		panic(err)
	}
	gpomWorkIcon, err := ioutil.ReadFile("assets/gpom-work.ico")
	if err != nil {
		panic(err)
	}
	gpomBreakIcon, err := ioutil.ReadFile("assets/gpom-break.ico")
	if err != nil {
		panic(err)
	}
	systray.SetIcon(gpomIcon)
	systray.SetTitle("GPom")
	go func() {
		for {
			select {
			case t := <-p.timer:
				systray.SetTitle(t)
			case tt := <-p.timerType:
				if tt == Work {
					fmt.Println("Work")
					systray.SetIcon(gpomWorkIcon)
				} else {
					systray.SetIcon(gpomBreakIcon)
				}
			}
		}
	}()

	systray.AddMenuItem("Pause", "")
	systray.AddMenuItem("Stop", "")
	systray.AddMenuItem("Statistics", "")
	systray.AddMenuItem("Settings", "")
	systray.AddMenuItem("Quit", "")
}

func (p *Pomo) onExit() {
}
