package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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

type commandFunc func(ctx context.Context) *cobra.Command

func registerCommand(cmd commandFunc) {
	cmds = append(cmds, cmd)
}

func executeCommand(ctx context.Context) {
	rootCmd := &cobra.Command{
		Use:   "rapid",
		Short: "Fetch and download",
		Long:  "Fetch and download a file from given url",
	}

	for _, command := range cmds {
		cmd := command(ctx)
		rootCmd.AddCommand(cmd)
	}

	rootCmd.Execute()
}

func download(ctx context.Context) *cobra.Command {
	const fetch = "http://localhost:9999/fetch"
	const download = "http://localhost:9999/%s/download/%s"

	cmd := &cobra.Command{
		Use:     "download",
		Aliases: []string{"d"},
		Example: "rapid download <url> | rapid d <url>",
		Short:   "Download a file from the given url",
		Run: func(cmd *cobra.Command, args []string) {
			provider, _ := cmd.Flags().GetString("provider")
			if provider == "" {
				provider = "default"
			}

			s := spinner.New(spinner.CharSets[26], 100*time.Millisecond)
			s.Prefix = "Fetching url"
			s.Suffix = "\n"

			s.Start()
			defer s.Stop()

			url := args[0]
			request := cliRequest{
				Url:      url,
				Provider: provider,
			}

			payload, err := json.Marshal(request)
			if err != nil {
				fmt.Println("Error marshalling request:", err)
				os.Exit(1)
				return
			}

			req, err := http.NewRequestWithContext(ctx, "POST", fetch, bytes.NewBuffer(payload))
			if err != nil {
				fmt.Println("Error preparing fetch request:", err.Error())
				os.Exit(1)
				return
			}

			req.Header.Add("Content-Type", "application/json")

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("Error creating fetch request:", err)
				os.Exit(1)
				return
			}

			defer res.Body.Close()

			var buffer bytes.Buffer
			if _, err := buffer.ReadFrom(res.Body); err != nil {
				fmt.Println(err)
				os.Exit(1)
				return
			}

			var result entry
			if err := json.Unmarshal(buffer.Bytes(), &result); err != nil {
				fmt.Println("Error unmarshalling buffer:", err)
				os.Exit(1)
				return
			}

			store(result.Id, result)

			fmt.Printf("Downloading %s (%s)\n\n\n", filepath.Base(result.Location), parseSize(result.Size))

			req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(download, ID, result.Id), nil)
			if err != nil {
				fmt.Println("Error preparing download request:", err.Error())
				os.Exit(1)
				return
			}

			res, err = http.DefaultClient.Do(req)
			if err != nil {
				log.Println("Error creating download request:", err)
				return
			}

			res.Body.Close()
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

func downloadWithOpen(ctx context.Context) *cobra.Command {
	// const server = "http://localhost:9999/download/cli"

	cmd := &cobra.Command{
		Use:     "open",
		Aliases: []string{"o"},
		Example: "rapid open <url> | rapid o <url>",
		Short:   "Download a file from the given URL using GUI",
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: implement this

		},
	}

	return cmd
}

func init() {
	registerCommand(downloadWithOpen)
	registerCommand(download)
}
