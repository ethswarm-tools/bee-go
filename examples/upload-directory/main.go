// Recursively upload a folder as a Swarm website.
//
// Usage:
//
//	go run ./examples/upload-directory <batch-id> <directory> [index-document]
//
// Defaults: index-document = "index.html". An empty string ("") omits
// the index document — the root path then returns 404 instead of
// serving any file. Once uploaded, the site is available at
// http://localhost:1633/bzz/<reference>/.
package main

import (
	"context"
	"fmt"
	"os"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: go run ./examples/upload-directory <batch-id> <directory> [index-document]")
		fmt.Println("example: go run ./examples/upload-directory 4a2... ./public index.html")
		os.Exit(1)
	}

	url := getenv("BEE_URL", "http://localhost:1633")

	batchID, err := swarm.BatchIDFromHex(os.Args[1])
	if err != nil {
		fail("invalid batch id: %v", err)
	}
	dir := os.Args[2]
	index := "index.html"
	if len(os.Args) >= 4 {
		index = os.Args[3]
	}

	client, err := bee.NewClient(url)
	if err != nil {
		fail("failed to create client: %v", err)
	}

	opts := &api.CollectionUploadOptions{
		IndexDocument: index,
	}

	fmt.Printf("Uploading directory %s using batch %s...\n", dir, batchID.Hex())
	if index != "" {
		fmt.Printf("Index document: %s\n", index)
	}
	fmt.Println()

	result, err := client.File.UploadCollection(context.Background(), batchID, dir, opts)
	if err != nil {
		fail("upload failed: %v", err)
	}

	fmt.Println("\nUpload successful!")
	fmt.Printf("Reference: %s\n", result.Reference.Hex())
	fmt.Printf("Browse at: %s/bzz/%s/\n", url, result.Reference.Hex())
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
