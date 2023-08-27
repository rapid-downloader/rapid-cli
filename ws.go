package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
)

var backgrounds = make([]BackgroundFunc, 0)

func registerBackground(r BackgroundFunc) {
	backgrounds = append(backgrounds, r)
}

func create(ctx context.Context) []Background {
	bgs := make([]Background, len(backgrounds))

	for _, background := range backgrounds {
		r := background(ctx)

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
		ctx context.Context
		progressBar
	}

	BackgroundFunc func(ctx context.Context) Background

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

func newServerListener(ctx context.Context) Background {
	return &serverListener{
		ctx:         ctx,
		progressBar: progressbar(),
	}
}

const ws = "ws://localhost:9999/ws/cli"

func (s *serverListener) Run() {
	cancel := s.ctx.Value("cancel").(context.CancelFunc)

	conn, res, err := websocket.DefaultDialer.DialContext(s.ctx, ws, nil)
	if err != nil {
		log.Fatalf("Error dialing websocket: %v. Status courlde %d", err, res.StatusCode)
	}

	defer conn.Close()
	defer truncateStore()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Error reading message:", err)
				break
			}

			var msg Message
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Println("Error unmarshalling message:", err)
				break
			}

			if msg.Done {
				cancel()
				break
			}

			s.update(msg.Index, msg.Downloaded, msg.Size)
		}
	}
}

func (s *serverListener) Close() {
	const stop = "http://localhost:9999/stop/%s"

	entry, ok := loadStored()
	if !ok {
		return
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf(stop, entry.Id), nil)
	if err != nil {
		log.Println("Error preparing stop request:", err)
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Error stopping download:", err)
		return
	}

	res.Body.Close()
}

func init() {
	registerBackground(newServerListener)
}
