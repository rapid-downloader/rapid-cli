package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
)

var backgrounds = make([]BackgroundFunc, 0)

func registerBackground(r BackgroundFunc) {
	backgrounds = append(backgrounds, r)
}

func create(done chan bool, signal chan os.Signal) []Background {
	bgs := make([]Background, len(backgrounds))

	for _, background := range backgrounds {
		r := background(done, signal)

		bgs = append(bgs, r)

		go r.Run()
	}

	return bgs
}

func shutdown(backgrounds []Background) {
	for _, background := range backgrounds {
		if closer, ok := background.(Closer); ok {
			closer.Close()
		}
	}
}

type (
	Background interface {
		Run()
	}

	Closer interface {
		Close()
	}

	serverListener struct {
		interrupt chan os.Signal
		done      chan bool
		progressBar
	}

	BackgroundFunc func(done chan bool, signal chan os.Signal) Background

	Message struct {
		ID         string `json:"id"`
		Index      int    `json:"index"`
		Downloaded int64  `json:"downloaded"`
		Progress   int64  `json:"progress"`
		Size       int64  `json:"size"`
		Done       bool   `json:"done"`
	}

	progressBar struct {
		mpb    *mpb.Progress
		barMap sync.Map
	}
)

func progressbar() progressBar {
	return progressBar{
		mpb:    mpb.New(),
		barMap: sync.Map{},
	}
}

func (p *progressBar) update(index int, downloaded int64, chunkSize int64) {
	i := fmt.Sprintf("%d", index)

	if val, ok := p.barMap.Load(i); ok {
		bar := val.(*mpb.Bar)
		bar.IncrBy(int(downloaded - bar.Current()))

		return
	}

	bar := p.mpb.AddBar(chunkSize,
		mpb.PrependDecorators(
			decor.CountersKiloByte("% .2f / % .2f"),
		),
		mpb.AppendDecorators(
			decor.AverageETA(decor.ET_STYLE_MMSS),
			decor.Name(" | "),
			decor.AverageSpeed(decor.UnitKB, "% .2f"),
		),
	)

	p.barMap.Store(i, bar)
}

func newServerListener(done chan bool, signal chan os.Signal) Background {
	return &serverListener{
		interrupt:   signal,
		done:        done,
		progressBar: progressbar(),
	}
}

const ws = "ws://localhost:3333/ws/cli"

func (s *serverListener) Run() {
	conn, res, err := websocket.DefaultDialer.Dial(ws, nil)
	if err != nil {
		log.Fatalf("Error dialing websocket: %v. Status courlde %d", err, res.StatusCode)
	}

	defer conn.Close()

	for {
		select {
		case <-s.done:
			return
		case <-s.interrupt:
			return
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Error reading message:", err)
				return
			}

			var msg Message
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Println("Error unmarshalling message:", err)
				return
			}

			if msg.Done {
				s.done <- true
				return
			}

			s.update(msg.Index, msg.Downloaded, msg.Size)
		}
	}
}

func init() {
	registerBackground(newServerListener)
}
