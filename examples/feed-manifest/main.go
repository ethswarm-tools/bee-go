// Create a feed manifest, then publish a couple of feed updates
// pointing at different content. Visiting /bzz/{feed-manifest-ref}/
// always returns the latest update — the URL stays stable across
// updates.
//
// Usage:
//
//	go run ./examples/feed-manifest <topic-string>
//
// Environment:
//   - BEE_URL        — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID   — usable postage batch hex (required)
//   - BEE_SIGNER_HEX — 32-byte hex private key (required)
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: feed-manifest <topic-string>")
	}
	topicStr := os.Args[1]

	url := getenv("BEE_URL", "http://localhost:1633")
	batchHex := os.Getenv("BEE_BATCH_ID")
	if batchHex == "" {
		return fmt.Errorf("BEE_BATCH_ID is required")
	}
	signerHex := os.Getenv("BEE_SIGNER_HEX")
	if signerHex == "" {
		return fmt.Errorf("BEE_SIGNER_HEX is required (32-byte hex)")
	}

	batchID, err := swarm.BatchIDFromHex(batchHex)
	if err != nil {
		return fmt.Errorf("invalid BEE_BATCH_ID: %w", err)
	}
	signer, err := swarm.PrivateKeyFromHex(signerHex)
	if err != nil {
		return fmt.Errorf("invalid BEE_SIGNER_HEX: %w", err)
	}
	owner := signer.PublicKey().Address()
	topic := swarm.TopicFromString(topicStr)

	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	ctx := context.Background()

	fmt.Printf("Creating feed manifest for (owner=%s, topic=%s)...\n", owner.Hex(), topic.Hex())
	feedManifest, err := client.File.CreateFeedManifest(ctx, batchID, owner, topic)
	if err != nil {
		return fmt.Errorf("create_feed_manifest: %w", err)
	}
	fmt.Printf("Feed manifest reference: %s\n", feedManifest.Hex())

	trimmed := strings.TrimRight(url, "/")
	stableURL := fmt.Sprintf("%s/bzz/%s/", trimmed, feedManifest.Hex())
	fmt.Printf("Stable URL (always latest): %s\n\n", stableURL)

	v1 := []byte("version 1: hello world")
	r1, err := client.File.UploadData(ctx, batchID, bytes.NewReader(v1), nil)
	if err != nil {
		return fmt.Errorf("upload v1: %w", err)
	}
	upd1, err := client.File.UpdateFeedWithReference(ctx, batchID, signer, topic, r1.Reference, nil)
	if err != nil {
		return fmt.Errorf("update v1: %w", err)
	}
	fmt.Printf("v1 content uploaded -> %s\n", r1.Reference.Hex())
	fmt.Printf("v1 feed update      -> %s\n", upd1.Reference.Hex())

	time.Sleep(1 * time.Second)

	v2 := []byte("version 2: hello swarm")
	r2, err := client.File.UploadData(ctx, batchID, bytes.NewReader(v2), nil)
	if err != nil {
		return fmt.Errorf("upload v2: %w", err)
	}
	upd2, err := client.File.UpdateFeedWithReference(ctx, batchID, signer, topic, r2.Reference, nil)
	if err != nil {
		return fmt.Errorf("update v2: %w", err)
	}
	fmt.Printf("\nv2 content uploaded -> %s\n", r2.Reference.Hex())
	fmt.Printf("v2 feed update      -> %s\n", upd2.Reference.Hex())

	fmt.Printf("\nAt %s, Bee resolves the feed pointer and serves\n", stableURL)
	fmt.Println("the LATEST update's content. The URL is stable across updates;")
	fmt.Println("readers never need to know the per-update reference.")
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
