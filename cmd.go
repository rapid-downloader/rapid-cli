package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

type (
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

	cliRequest struct {
		Url         string   `json:"url"`
		Client      string   `json:"client"`
		Provider    string   `json:"provider"`
		ContentType string   `json:"contentType"`
		UserAgent   string   `json:"userAgent"`
		Cookies     []cookie `json:"cookies"`
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

var cmds = make([]commandFunc, 0)

type commandFunc func(done chan bool) *cobra.Command

func registerCommand(cmd commandFunc) {
	cmds = append(cmds, cmd)
}

func executeCommand(done chan bool) {
	rootCmd := &cobra.Command{
		Use:   "rapid",
		Short: "Fetch and download",
		Long:  "Fetch and download a file from given url",
	}

	for _, command := range cmds {
		cmd := command(done)
		rootCmd.AddCommand(cmd)
	}

	rootCmd.Execute()
}

func download(done chan bool) *cobra.Command {
	const fetch = "http://localhost:3333/fetch"
	const download = "http://localhost:3333/cli/download/%s"

	cmd := &cobra.Command{
		Use:     "download",
		Aliases: []string{"d"},
		Example: "rapid download <url> | rapid d <url>",
		Short:   "Download a file from the given url",
		Run: func(cmd *cobra.Command, args []string) {
			s := spinner.New(spinner.CharSets[26], 100*time.Millisecond)
			s.Prefix = "Fetching url"
			s.Suffix = "\n"

			s.Start()
			defer s.Stop()

			url := args[0]
			request := cliRequest{
				Url: url,
			}

			payload, err := json.Marshal(request)
			if err != nil {
				log.Println("Error marshalling request:", err)
				return
			}

			res, err := http.Post(fetch, "application/json", bytes.NewBuffer(payload))
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

			fmt.Printf("Downloading %s (%s)\n\n\n", filepath.Base(result.Data.Location), parseSize(result.Data.Size))
			req, err := http.Get(fmt.Sprintf(download, result.Data.Id))
			if err != nil {
				log.Println("Error creating request:", err)
				return
			}

			req.Body.Close()
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

func downloadWithOpen(done chan bool) *cobra.Command {
	// const server = "http://localhost:3333/download/cli"

	cmd := &cobra.Command{
		Use:     "open",
		Aliases: []string{"o"},
		Example: "rapid open <url> | rapid o <url>",
		Short:   "Download a file from the given URL using GUI",
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: implement this

			done <- true
		},
	}

	return cmd
}

func init() {
	registerCommand(downloadWithOpen)
	registerCommand(download)
}
