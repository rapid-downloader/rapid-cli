package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
)

type progressBar struct {
	ID       string  `json:"id"`
	Index    int     `json:"index"`
	Progress float64 `json:"progress"`
	Done     bool    `json:"bool"`
}

func (p *progressBar) String() string {
	var buff bytes.Buffer

	buff.WriteString(fmt.Sprintf("Chunk %d - %v downloaded", p.Index, p.Progress))

	return buff.String()
}

type entry struct {
	Id               string `json:"id"`
	Name             string `json:"name"`
	Location         string `json:"location"`
	Size             int64  `json:"size"`
	Filetype         string `json:"filetype"`
	URL              string `json:"url"`
	Resumable        bool   `json:"resumable"`
	ChunkLen         int    `json:"chunkLen"`
	DownloadProvider string `json:"downloadProvider"`
}

func (e *entry) String() string {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("ID: %v\n", e.Id))
	buffer.WriteString(fmt.Sprintf("Name: %v\n", e.Name))
	buffer.WriteString(fmt.Sprintf("Location: %v\n", e.Location))
	buffer.WriteString(fmt.Sprintf("Size: %v\n", e.Size))
	buffer.WriteString(fmt.Sprintf("Filetype: %v\n", e.Filetype))
	buffer.WriteString(fmt.Sprintf("Resumable: %v\n", e.Resumable))
	buffer.WriteString(fmt.Sprintf("ChunkLen: %v\n", e.ChunkLen))

	return buffer.String()
}

type responseBody struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

type body struct {
	Url string `json:"url"`
}

type Map map[string]interface{}

const serverWs = "ws://localhost:3300/ws/cli"
const server = "http://localhost:3300/download/cli"

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	conn, res, err := websocket.DefaultDialer.Dial(serverWs, nil)
	if err != nil {
		log.Fatalf("Error connecting websocket: %v. Status code %d", err, res.StatusCode)
	}

	defer conn.Close()

	body := body{
		Url: "https://link.testfile.org/PDF50MB",
	}

	payload, err := json.Marshal(body)
	if err != nil {
		log.Fatal("Error marshalling body:", err)
	}

	response, err := http.DefaultClient.Post(server, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Fatal("Error performing http post:", err)
	}

	defer response.Body.Close()

	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(response.Body); err != nil {
		log.Fatal(err)
	}

	var result responseBody
	if err := json.Unmarshal(buffer.Bytes(), &result); err != nil {
		log.Fatal("Error unmarshalling buffer:", err)
	}

	if entry, ok := result.Data.(entry); ok {
		log.Println(entry.String())
	}

	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case <-interrupt:
				return
			default:
				_, message, err := conn.ReadMessage()

				if err != nil {
					log.Println("Error reading message:", err)
					return
				}

				var msg Map
				if err := json.Unmarshal(message, &msg); err != nil {
					log.Println("Error unmarshalling message:", err)
					return
				}

				if _, ok := msg["done"]; ok {
					done <- struct{}{}
					return
				}

			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("Shutting down...")
			if err := conn.WriteMessage(websocket.TextMessage, []byte("G	ood bye...")); err != nil {
				log.Println("Error sending message:", err)
				return
			}

			return
		}
	}
}
