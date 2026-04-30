package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ethersphere/bee-go"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

func main() {
	// We read the batch ID and file name from command line arguments for convenience
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <batch-id> [file-path]")
		fmt.Println("Example: go run main.go 4a2... image.png")
		os.Exit(1)
	}

	batchID, err := swarm.BatchIDFromHex(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid batch ID: %v", err)
	}
	
	// Default to "image.png" if no specific file was provided
	filePath := "image.png"
	if len(os.Args) >= 3 {
		filePath = os.Args[2]
	}

	// Connect to the main Bee API (port 1633)
	client, err2 := bee.NewClient("http://localhost:1633")
	if err2 != nil {
		log.Fatalf("Failed to create client: %v", err2)
	}

	// 1. Open the picture file
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open %s. Make sure the file exists! Error: %v", filePath, err)
	}
	defer file.Close()

	fmt.Printf("Uploading %s using batch %s...\n", filePath, batchID.Hex())

	// 2. Upload the File
	// Assuming it's a PNG image, we set the Content-Type to "image/png"
	// Signature: UploadFile(ctx, batchID, reader, name, contentType, uploadOptions)
	ref, err := client.File.UploadFile(context.Background(), batchID, file, filePath, "image/png", nil)
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	fmt.Println("Upload successful!")
	fmt.Printf("Reference: %s\n", ref.Reference.Hex())

	// 3. Provide the retrieval link
	// Files uploaded to `/bzz` can be retrieved at `/bzz/{reference}`
	downloadLink := fmt.Sprintf("http://localhost:1633/bzz/%s", ref.Reference.Hex())
	fmt.Printf("\nYou can view your picture in the browser at:\n%s\n", downloadLink)
}
