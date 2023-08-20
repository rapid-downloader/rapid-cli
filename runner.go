package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/schollz/progressbar/v3"
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
		ws        *websocket.Conn
		interrupt chan os.Signal
		done      chan bool
		bar       progressBar
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
		*sync.Mutex
		bar map[string]*progressbar.ProgressBar
	}
)

func newProgressBar() progressBar {
	return progressBar{
		&sync.Mutex{},
		make(map[string]*progressbar.ProgressBar),
	}
}

func (b *progressBar) update(index int, bytes int64, maxBytes int64) {
	b.Lock()
	defer b.Unlock()

	i := fmt.Sprintf("%d", index)

	if progress, ok := b.bar[i]; ok {
		progress.Set64(bytes)

		return
	}

	bar := progressbar.DefaultBytes(maxBytes)
	bar.Set64(bytes)

	b.bar[i] = bar
}

const ws = "ws://localhost:3333/ws/cli"

func newServerListener(done chan bool, signal chan os.Signal) Background {
	conn, res, err := websocket.DefaultDialer.Dial(ws, nil)
	if err != nil {
		log.Fatalf("Error dialing websocket: %v. Status courlde %d", err, res.StatusCode)
	}

	return &serverListener{
		ws:        conn,
		interrupt: signal,
		done:      done,
		bar:       newProgressBar(),
	}
}

func (s *serverListener) Run() {
	for {
		select {
		case <-s.done:
			return
		case <-s.interrupt:
			return
		default:
			_, message, err := s.ws.ReadMessage()

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
			}

			s.bar.update(msg.Index, msg.Downloaded, msg.Size)
		}
	}
}

func (s *serverListener) Close() {
	s.ws.Close()
}

func init() {
	registerBackground(newServerListener)
}
