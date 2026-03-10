package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ethersphere/bee-go"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

func main() {
	// We require the Swarm reference (hash) to download
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <reference> [output-filename]")
		fmt.Println("Example: go run main.go 4a2... downloaded_image.png")
		os.Exit(1)
	}

	reference := os.Args[1]
	
	// Default to "downloaded.png" if no output filename is provided
	outputFilename := "downloaded.png"
	if len(os.Args) >= 3 {
		outputFilename = os.Args[2]
	}

	// Connect to the main Bee API (port 1633)
	client, err := bee.NewClient("http://localhost:1633")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Printf("Downloading reference %s...\n", reference)

	// 1. Download the File
	// DownloadFile(ctx, reference) returns an io.ReadCloser and the Content-Type
	ref := swarm.Reference{Value: reference}
	reader, contentType, err := client.File.DownloadFile(context.Background(), ref)
	if err != nil {
		log.Fatalf("Download failed: %v", err)
	}
	defer reader.Close()

	fmt.Printf("File found! Content-Type: %s\n", contentType)

	// 2. Create the local file
	outFile, err := os.Create(outputFilename)
	if err != nil {
		log.Fatalf("Failed to create local file %s: %v", outputFilename, err)
	}
	defer outFile.Close()

	// 3. Save the data to disk
	bytesWritten, err := io.Copy(outFile, reader)
	if err != nil {
		log.Fatalf("Failed to save data: %v", err)
	}

	fmt.Printf("Successfully downloaded and saved to %s (%d bytes)\n", outputFilename, bytesWritten)
}
