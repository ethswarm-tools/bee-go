// Write three feed updates, then walk indexes 0..N and print the
// timeline. Demonstrates that feed updates are addressable per-index
// via SOC reads, in addition to the "latest" lookup.
//
// Usage:
//
//	go run ./examples/feed-history <topic-string>
//
// Environment:
//   - BEE_URL        — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID   — usable postage batch hex (required)
//   - BEE_SIGNER_HEX — 32-byte hex private key (required)
package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"time"
	"unicode/utf8"

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
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: feed-history <topic-string>")
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

	start, err := client.File.FindNextIndex(ctx, owner, topic)
	if err != nil {
		return fmt.Errorf("find_next_index: %w", err)
	}
	fmt.Printf("Starting feed write at index %d (existing latest+1).\n", start)

	for i := uint64(0); i < 3; i++ {
		idx := start + i
		payload := []byte(fmt.Sprintf("entry #%d at index %d", i, idx))
		if _, err := client.File.UpdateFeedWithIndex(ctx, batchID, signer, topic, idx, payload); err != nil {
			return fmt.Errorf("update_feed_with_index %d: %w", idx, err)
		}
		fmt.Printf("  wrote index %d: %q\n", idx, string(payload))
		time.Sleep(1 * time.Second)
	}
	last := start + 2

	fmt.Printf("\nFeed history (indexes 0..=%d):\n", last)
	reader := client.File.MakeSOCReader(owner)
	for i := uint64(0); i <= last; i++ {
		id, err := file.MakeFeedIdentifier(topic, i)
		if err != nil {
			return fmt.Errorf("make_feed_identifier %d: %w", i, err)
		}
		soc, err := reader.Download(ctx, id)
		if err != nil {
			fmt.Printf("  index %3d: missing (%v)\n", i, err)
			continue
		}
		if len(soc.Payload) < 8 {
			fmt.Printf("  index %3d: <bad payload>\n", i)
			continue
		}
		ts := binary.BigEndian.Uint64(soc.Payload[:8])
		body := soc.Payload[8:]
		if utf8.Valid(body) {
			fmt.Printf("  index %3d  ts=%d  %q\n", i, ts, string(body))
		} else {
			fmt.Printf("  index %3d  ts=%d  (%d bytes binary)\n", i, ts, len(body))
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
