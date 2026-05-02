// Build a Mantaray manifest from a list of entries, hash it offline,
// then add a new entry, hash again, and upload via UploadCollectionEntries.
// Demonstrates that adding a file changes the manifest reference
// deterministically.
//
// Usage:
//
//	go run ./examples/manifest-add-file
//
// Environment:
//   - BEE_URL      — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID — usable postage batch (required)
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/file"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	url := getenv("BEE_URL", "http://localhost:1633")
	batchHex := os.Getenv("BEE_BATCH_ID")
	if batchHex == "" {
		return fmt.Errorf("BEE_BATCH_ID is required")
	}
	batchID, err := swarm.BatchIDFromHex(batchHex)
	if err != nil {
		return fmt.Errorf("invalid BEE_BATCH_ID: %w", err)
	}

	entries := []file.CollectionEntry{
		{Path: "index.html", Data: []byte("<h1>hello</h1>")},
		{Path: "about.html", Data: []byte("<h1>about</h1>")},
	}
	root1, err := file.HashCollectionEntries(entries)
	if err != nil {
		return fmt.Errorf("hash v1: %w", err)
	}
	fmt.Printf("Initial manifest (%d entries) → %s\n", len(entries), root1.Hex())

	entries = append(entries, file.CollectionEntry{
		Path: "contact.html",
		Data: []byte("<h1>contact</h1>"),
	})
	root2, err := file.HashCollectionEntries(entries)
	if err != nil {
		return fmt.Errorf("hash v2: %w", err)
	}
	fmt.Printf("With contact.html (%d entries) → %s\n", len(entries), root2.Hex())

	if root1.Hex() == root2.Hex() {
		return fmt.Errorf("manifest root did not change after adding a file")
	}
	fmt.Println("(root changed — adding a file mutates the manifest deterministically)")

	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	result, err := client.File.UploadCollectionEntries(context.Background(), batchID, entries, nil)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	fmt.Printf("\nUploaded → %s\n", result.Reference.Hex())
	if result.Reference.Hex() == root2.Hex() {
		fmt.Println("matches offline hash ✓")
	} else {
		return fmt.Errorf("uploaded reference %s != offline root %s", result.Reference.Hex(), root2.Hex())
	}

	trimmed := strings.TrimRight(url, "/")
	fmt.Printf("\nBrowse at: %s/bzz/%s/index.html\n", trimmed, result.Reference.Hex())
	fmt.Printf("           %s/bzz/%s/contact.html\n", trimmed, result.Reference.Hex())
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
