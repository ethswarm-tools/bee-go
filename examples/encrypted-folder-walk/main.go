// encrypted-folder-walk uploads a small set of files as a collection,
// then downloads the manifest's root chunk and walks the fork tree
// recursively, fetching each child chunk and stitching the file paths
// back together.
//
// Bee builds Mantaray manifests as a tree of chunks: the root only
// holds the first byte of each file path, and longer prefixes spill
// into child chunks. To reach a leaf you have to follow each fork's
// child address until you hit a node whose target_address is set.
//
// Note on the name: an earlier version used encrypt=true, but Bee's
// /chunks endpoint only accepts 32-byte addresses, so there is no
// clean way to download the encrypted manifest chunk itself without
// client-side AES-CTR decryption. We keep the filename for continuity
// but demonstrate the cleaner unencrypted variant; see
// encrypted-upload for the encryption round-trip.
//
// Usage:
//
//	go run ./examples/encrypted-folder-walk
//
// Environment:
//   - BEE_URL      — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID — usable postage batch (required)
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

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
	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	ctx := context.Background()

	// 1. Upload a few short files as a collection. Bee builds the
	//    manifest as a chunk tree where each filename is split across
	//    one or more chunks.
	entries := []file.CollectionEntry{
		{Path: "readme.txt", Data: []byte("hello from the manifest walker")},
		{Path: "notes.md", Data: []byte("# notes\n- one\n- two\n")},
	}
	result, err := client.File.UploadCollectionEntries(ctx, batchID, entries, nil)
	if err != nil {
		return fmt.Errorf("upload_collection_entries: %w", err)
	}
	manifestRef := result.Reference
	fmt.Printf("Manifest reference: %s (%d bytes)\n", manifestRef.Hex(), manifestRef.Len())

	// 2. Walk the manifest tree recursively. Each node we parse may
	//    expose a target_address (a leaf — done) and/or more forks
	//    pointing at child chunks we need to fetch.
	leaves := map[string]string{}
	if err := walk(ctx, client, manifestRef, nil, leaves); err != nil {
		return err
	}
	if len(leaves) == 0 {
		return fmt.Errorf("manifest had no leaves")
	}
	fmt.Println("\nManifest entries:")
	for path, leafHex := range leaves {
		fmt.Printf("  - %-24s → %s (%d hex chars)\n", path, leafHex, len(leafHex))
	}

	// 3. Download each leaf via /bytes (manifest-resolved by Bee).
	for path, leafHex := range leaves {
		leaf, err := swarm.ReferenceFromHex(leafHex)
		if err != nil {
			return fmt.Errorf("invalid leaf reference for %s: %w", path, err)
		}
		body, err := client.File.DownloadData(ctx, leaf, nil)
		if err != nil {
			return fmt.Errorf("download %s: %w", path, err)
		}
		got, err := io.ReadAll(body)
		body.Close()
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if utf8.Valid(got) {
			fmt.Printf("  fetched %s: %q\n", path, string(got))
		} else {
			fmt.Printf("  fetched %s: %d bytes (binary)\n", path, len(got))
		}
	}
	return nil
}

func walk(ctx context.Context, client *bee.Client, ref swarm.Reference,
	pathSoFar []byte, out map[string]string) error {
	raw, err := client.File.DownloadChunk(ctx, ref, nil)
	if err != nil {
		return fmt.Errorf("download_chunk: %w", err)
	}
	body := raw
	if len(raw) >= 8 {
		body = raw[8:]
	}
	node, err := manifest.Unmarshal(body, ref.Raw())
	if err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	if !manifest.IsNullAddress(node.TargetAddress) {
		leaf, err := swarm.NewReference(node.TargetAddress)
		if err != nil {
			return fmt.Errorf("entry ref: %w", err)
		}
		out[string(pathSoFar)] = leaf.Hex()
	}
	for _, fork := range node.Forks {
		if manifest.IsNullAddress(fork.Node.SelfAddress) {
			continue
		}
		childRef, err := swarm.NewReference(fork.Node.SelfAddress)
		if err != nil {
			return fmt.Errorf("child ref: %w", err)
		}
		next := append([]byte{}, pathSoFar...)
		next = append(next, fork.Prefix...)
		if err := walk(ctx, client, childRef, next, out); err != nil {
			return err
		}
	}
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
