package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

	progress struct {
		ID         string `json:"id"`
		Index      int    `json:"index"`
		Downloaded int64  `json:"downloaded"`
		Progress   int64  `json:"progress"`
		Size       int64  `json:"size"`
		Done       bool   `json:"done"`
	}
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func randID(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)

	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}

var ID = randID(5)

const ws = "ws://localhost:9999/ws/%s"

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

	req, err := http.NewRequest("PUT", fmt.Sprintf(stop, entry.ID), nil)
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

	conn, res, err := websocket.DefaultDialer.DialContext(ctx, fmt.Sprintf(ws, ID), nil)
	if err != nil {
		log.Fatalf("Error dialing websocket: %v. Status courlde %d", err, res.StatusCode)
		return
	}

	executeCommand(ctx)

	progressBar := progressbar()
	onError := make(chan bool)

	go func() {
		for {
			select {
			case <-onError:
				return
			default:
				_, message, err := conn.ReadMessage()
				if err != nil {
					fmt.Println(err.Error())
					return
				}

				var progress progress
				if err := json.Unmarshal(message, &progress); err != nil {
					fmt.Println("Error unmarshalling message:", err)
					return
				}

				if progress.Done {
					truncateStore()
					cancel()
					return
				}

				progressBar.update(progress.Index, progress.Downloaded, progress.Size)
			}
		}
	}()

	ping := time.Tick(time.Second)

	go func() {
		for {
			select {
			case <-ping:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					fmt.Println(err.Error())
					onError <- true
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-onError:
			closeConn(conn)
			return
		case <-ctx.Done():
			stopDownload()
			closeConn(conn)
			return
		case <-interrupt:
			cancel()
		}
	}
}

func closeConn(conn *websocket.Conn) {
	if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		fmt.Println("Error sending close signal to server:", err)
		return
	}
}
