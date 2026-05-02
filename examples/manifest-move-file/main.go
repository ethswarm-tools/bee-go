// Move a file's path within a manifest by removing the fork at the
// old path and adding it at the new path, preserving the file's
// content reference. Demonstrates Mantaray's offline AddFork /
// RemoveFork primitives.
//
// Usage:
//
//	go run ./examples/manifest-move-file
//
// Environment:
//   - BEE_URL      — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID — usable postage batch (required)
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/file"
	"github.com/ethswarm-tools/bee-go/pkg/manifest"
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

	logo, _ := swarm.ReferenceFromHex(strings.Repeat("aa", 32))
	about, _ := swarm.ReferenceFromHex(strings.Repeat("bb", 32))
	style, _ := swarm.ReferenceFromHex(strings.Repeat("cc", 32))

	v1 := manifest.New()
	v1.AddFork([]byte("images/logo.png"), logo, nil)
	v1.AddFork([]byte("about.html"), about, nil)
	v1.AddFork([]byte("style.css"), style, nil)
	rootV1, err := v1.CalculateSelfAddress()
	if err != nil {
		return fmt.Errorf("v1 self_address: %w", err)
	}
	fmt.Println("v1 manifest:")
	for _, n := range v1.Collect() {
		fmt.Printf("  - %s\n", n.FullPathString())
	}
	fmt.Printf("  root: %s\n", hex.EncodeToString(rootV1))

	v2 := manifest.New()
	v2.AddFork([]byte("assets/logo.png"), logo, nil)
	v2.AddFork([]byte("about.html"), about, nil)
	v2.AddFork([]byte("style.css"), style, nil)
	rootV2, err := v2.CalculateSelfAddress()
	if err != nil {
		return fmt.Errorf("v2 self_address: %w", err)
	}
	fmt.Println("\nv2 manifest (logo.png moved):")
	for _, n := range v2.Collect() {
		fmt.Printf("  - %s\n", n.FullPathString())
	}
	fmt.Printf("  root: %s\n", hex.EncodeToString(rootV2))

	// Surgical: remove + add on a fresh build (cloning is awkward in Go).
	surgical := manifest.New()
	surgical.AddFork([]byte("images/logo.png"), logo, nil)
	surgical.AddFork([]byte("about.html"), about, nil)
	surgical.AddFork([]byte("style.css"), style, nil)
	if err := surgical.RemoveFork([]byte("images/logo.png")); err != nil {
		return fmt.Errorf("remove_fork: %w", err)
	}
	surgical.AddFork([]byte("assets/logo.png"), logo, nil)
	surgicalRoot, err := surgical.CalculateSelfAddress()
	if err != nil {
		return fmt.Errorf("surgical self_address: %w", err)
	}
	fmt.Printf("\nsurgical (RemoveFork + AddFork) → same root as rebuild? %t\n",
		hex.EncodeToString(surgicalRoot) == hex.EncodeToString(rootV2))

	// To make the moved manifest live on Bee, upload via
	// UploadCollectionEntries with the new layout. Note: this re-uploads
	// the underlying file bytes too — bee-go and bee-rs don't yet expose
	// in-place mutation without re-uploading leaves.
	entries := []file.CollectionEntry{
		{Path: "assets/logo.png", Data: []byte("<png bytes>")},
		{Path: "about.html", Data: []byte("<h1>about</h1>")},
		{Path: "style.css", Data: []byte("body { color: red }")},
	}
	offlineRoot, err := file.HashCollectionEntries(entries)
	if err != nil {
		return fmt.Errorf("hash entries: %w", err)
	}
	fmt.Printf("\nOffline hash for upload entries: %s\n", offlineRoot.Hex())

	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	result, err := client.File.UploadCollectionEntries(context.Background(), batchID, entries, nil)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	trimmed := strings.TrimRight(url, "/")
	fmt.Printf("Uploaded   → %s\n", result.Reference.Hex())
	fmt.Printf("Browse at: %s/bzz/%s/assets/logo.png\n", trimmed, result.Reference.Hex())
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
