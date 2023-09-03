package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
)

type (
	progressBar struct {
		mpb    *mpb.Progress
		barMap sync.Map
	}

	serverListener struct {
		ctx context.Context
		ws  *websocket.Conn
		progressBar
	}

	progress struct {
		ID         string `json:"id"`
		Index      int    `json:"index"`
		Downloaded int64  `json:"downloaded"`
		Progress   int64  `json:"progress"`
		Size       int64  `json:"size"`
		Done       bool   `json:"done"`
	}
)

const ws = "ws://localhost:9999/ws/cli"

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

func stopDownload() {
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

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, []os.Signal{syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGSTOP, os.Interrupt}...)

	ctx, cancel := context.WithCancel(context.Background())

	conn, res, err := websocket.DefaultDialer.DialContext(ctx, ws, nil)
	if err != nil {
		log.Fatalf("Error dialing websocket: %v. Status courlde %d", err, res.StatusCode)
		return
	}

	executeCommand(ctx)

	progressBar := progressbar()

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Error reading message:", err)
				break
			}

			var progress progress
			if err := json.Unmarshal(message, &progress); err != nil {
				log.Println("Error unmarshalling message:", err)
				break
			}

			if progress.Done {
				truncateStore()
				cancel()
				break
			}

			progressBar.update(progress.Index, progress.Downloaded, progress.Size)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			closeConn(ctx, conn)
			return
		case <-interrupt:
			stopDownload()
			closeConn(ctx, conn)
			return
		}
	}
}

func closeConn(ctx context.Context, conn *websocket.Conn) {
	if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		log.Println("Error sending close signal to server:", err)
		return
	}

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
	}
}
