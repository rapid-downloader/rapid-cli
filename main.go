package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"github.com/goccy/go-json"
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

type body struct {
	Url string `json:"url"`
}

const serverWs = "ws://localhost:3300/ws/cli"
const server = "http://localhost:3300/download/cli"

func main() {
	// interrupt := make(chan os.Signal, 1)
	// signal.Notify(interrupt, os.Interrupt)

	// conn, response, err := websocket.DefaultDialer.Dial(serverWs, nil)
	// if err != nil {
	// 	log.Fatalf("Error connecting websocket: %v. Status code %d", err, response.StatusCode)
	// }

	// defer conn.Close()

	body := body{
		Url: "https://rr5---sn-vgqsknz7.googlevideo.com/videoplayback?expire=1692478697&ei=idjgZJFRmeWKvg_zwZ6ICw&ip=212.102.59.151&id=o-AKd6YtvaPJyKVzEWnEpnDJlSTUtRk__kuNbuQM-HEQuR&itag=22&source=youtube&requiressl=yes&mh=1P&mm=31%2C26&mn=sn-vgqsknz7%2Csn-p5qlsn6l&ms=au%2Conr&mv=m&mvi=5&pl=26&initcwndbps=642500&spc=UWF9f9Ji9MprP7GV-lx_MwWnbEO7xYE&vprv=1&svpuc=1&mime=video%2Fmp4&cnr=14&ratebypass=yes&dur=8673.790&lmt=1682606470933426&mt=1692456802&fvip=5&fexp=24007246%2C24363392&c=ANDROID&txp=5318224&sparams=expire%2Cei%2Cip%2Cid%2Citag%2Csource%2Crequiressl%2Cspc%2Cvprv%2Csvpuc%2Cmime%2Ccnr%2Cratebypass%2Cdur%2Clmt&sig=AOq0QJ8wRgIhAI7PbAZk-M6T2h7v-mGhjKg3oeAMhoERXYUF4zmMRIjsAiEAyb0p5WHTsxVfenCbrbH8p5mU4PgR_GIFcdCeD0iFbwI%3D&lsparams=mh%2Cmm%2Cmn%2Cms%2Cmv%2Cmvi%2Cpl%2Cinitcwndbps&lsig=AG3C_xAwRAIgR6rk9MR9p3bEmTClFzDO_oSzZoybxaExcVF6K8SPXDsCIGykg36QfBJGwo5lDrabitVv_DTR3pGcc2IJxu4_jbWw&title=Mastering%20WebSockets%20With%20Go%20-%20An%20in-depth%20tutorial",
	}

	payload, err := json.Marshal(body)
	if err != nil {
		log.Fatal("Error marshalling body:", err)
	}

	res, err := http.DefaultClient.Post(server, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Fatal("Error performing http post:", err)
	}

	defer res.Body.Close()

	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(res.Body); err != nil {
		log.Fatal(err)
	}

	var result interface{}
	if err := json.Unmarshal(buffer.Bytes(), &result); err != nil {
		log.Fatal("Error unmarshalling buffer:", err)
	}

	log.Println(result)

	// done := make(chan struct{})

	// go func() {
	// 	for {
	// 		select {
	// 		case <-interrupt:
	// 			return
	// 		default:
	// 			_, message, err := conn.ReadMessage()

	// 			if err != nil {
	// 				log.Println("Error reading message:", err)
	// 				return
	// 			}

	// 			var progress progressBar
	// 			if err := json.Unmarshal(message, &progress); err != nil {
	// 				log.Println("Error unmarshalling message:", err)
	// 				return
	// 			}

	// 			log.Println(progress)

	// 			if progress.Done {
	// 				done <- struct{}{}
	// 				return
	// 			}
	// 		}
	// 	}
	// }()

	// ticker := time.NewTicker(time.Second)
	// defer ticker.Stop()

	// for {
	// 	select {
	// 	case <-done:
	// 		return
	// 	case <-interrupt:
	// 		log.Println("Shutting down...")
	// 		if err := conn.WriteMessage(websocket.TextMessage, []byte("G	ood bye...")); err != nil {
	// 			log.Println("Error sending message:", err)
	// 			return
	// 		}

	// 		return
	// 	}
	// }
}
