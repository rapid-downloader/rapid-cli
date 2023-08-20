package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

type (
	cliRequest struct {
		URL string `json:"url"`
	}

	cookie struct {
		Name     string    `json:"name"`
		Value    string    `json:"value"`
		Path     string    `json:"path"`
		Domain   string    `json:"domain"`
		Expires  time.Time `json:"expirationDate"`
		Secure   bool      `json:"secure"`
		HttpOnly bool      `json:"httpOnly"`
		SameSite string    `json:"sameSite"`
	}

	openRequest struct {
		Url         string   `json:"url"`
		ContentType string   `json:"contentType"`
		UserAgent   string   `json:"userAgent"`
		cookies     []cookie `json:"cookies"`
	}

	cliResponse struct {
		Status string `json:"status"`
		Data   entry  `json:"data"`
	}

	entry struct {
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
)

func (e *entry) String() string {
	var buffer bytes.Buffer

	buffer.WriteString("==============================\n")
	buffer.WriteString(fmt.Sprintf("ID: %v\n", e.Id))
	buffer.WriteString(fmt.Sprintf("Name: %v\n", e.Name))
	buffer.WriteString(fmt.Sprintf("Location: %v\n", e.Location))
	buffer.WriteString(fmt.Sprintf("Size: %v\n", e.Size))
	buffer.WriteString(fmt.Sprintf("Filetype: %v\n", e.Filetype))
	buffer.WriteString(fmt.Sprintf("Resumable: %v\n", e.Resumable))
	buffer.WriteString(fmt.Sprintf("Total Chunks: %v\n", e.ChunkLen))
	buffer.WriteString("==============================\n")

	return buffer.String()
}

func download() *cobra.Command {
	const server = "http://localhost:3333/download/cli"

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Fetch and download",
		Long:  "Fetch and download a file from given URL",
		Run: func(cmd *cobra.Command, args []string) {
			url := args[0]
			request := cliRequest{
				URL: url,
			}

			payload, err := json.Marshal(request)
			if err != nil {
				log.Println("Error marshalling request:", err)
				return
			}

			fmt.Println("Fetching url...")
			res, err := http.Post(server, "application/json", bytes.NewBuffer(payload))
			if err != nil {
				log.Println("Error marshalling request:", err)
				return
			}

			defer res.Body.Close()

			var buffer bytes.Buffer
			if _, err := buffer.ReadFrom(res.Body); err != nil {
				log.Fatal(err)
			}

			var result cliResponse
			if err := json.Unmarshal(buffer.Bytes(), &result); err != nil {
				log.Fatal("Error unmarshalling buffer:", err)
			}

			fmt.Println(result.Data.String())
		},
	}

	return cmd
}

func downloadWithOpen() *cobra.Command {
	// TODO: implement this
	return nil
}
