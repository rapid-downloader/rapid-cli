package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/briandowns/spinner"
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
			s := spinner.New(spinner.CharSets[26], 100*time.Millisecond)
			s.Prefix = "Fetching url"
			s.Suffix = "\n"

			s.Start()
			defer s.Stop()

			url := args[0]
			request := cliRequest{
				URL: url,
			}

			payload, err := json.Marshal(request)
			if err != nil {
				log.Println("Error marshalling request:", err)
				return
			}

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

			fmt.Printf("Downloading %s (%s)\n", result.Data.Name, parseSize(result.Data.Size))
		},
	}

	return cmd
}

func parseSize(size int64) string {
	const KB = 1024
	const MB = KB * KB
	const GB = MB * KB

	if size < KB {
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	}

	if size > KB && size < MB {
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	}

	if size > KB && size < GB {
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	}

	if size > GB {
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	}

	return "0 KB"
}

// TODO: add command registry

func downloadWithOpen() *cobra.Command {
	// TODO: implement this
	return nil
}
