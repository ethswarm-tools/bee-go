// swarm-relay is a single-batch upload gateway with persisted bucket
// tracking. Foundation for any "free-tier" relay or shared-batch
// service: hold one batch + signer, accept user uploads, refuse them
// pre-emptively when the local Stamper says a bucket is full.
//
// Each `relay` invocation:
//  1. Loads (or initialises) Stamper state from .swarm-relay-state.json.
//  2. Hashes the input file via FileChunker to enumerate every chunk
//     address that will land on Bee.
//  3. For each address, calls Stamper.Stamp — sign + bucket
//     increment. Failure here means the batch can no longer cover
//     this file; we abort *before* uploading.
//  4. Uploads the file via UploadData (Bee re-stamps internally; the
//     local stamps act as predictive bookkeeping).
//  5. Persists the new bucket state.
//
// Note: bee-go's Stamper is in-memory only — depth/buckets are
// rebuilt from JSON each run. (bee-rs has from_state for richer
// resume; the JSON snapshot here works the same way.)
//
// Usage:
//
//	swarm-relay init                # initialise state
//	swarm-relay relay <local-file>  # ingest one file
//	swarm-relay stats               # batch utilization snapshot
//
// Wrap this binary in any HTTP server (net/http, gin, …) to expose a
// `POST /upload` endpoint — the relay logic is the interesting part.
//
// Environment:
//   - BEE_URL        — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID   — usable postage batch (required for init)
//   - BEE_SIGNER_HEX — 32-byte hex private key (required for init)
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/postage"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

const (
	stateFile  = ".swarm-relay-state.json"
	numBuckets = 65536
)

type relayState struct {
	BatchID       string   `json:"batch_id"`
	SignerHex     string   `json:"signer_hex"`
	Depth         uint8    `json:"depth"`
	Buckets       []uint32 `json:"buckets"`
	UploadedFiles uint32   `json:"uploaded_files"`
	UploadedBytes uint64   `json:"uploaded_bytes"`
	RejectedFiles uint32   `json:"rejected_files"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	url := getenv("BEE_URL", "http://localhost:1633")
	args := os.Args[1:]
	if len(args) == 0 {
		return fmt.Errorf("usage: swarm-relay <init|relay|stats|reset>")
	}
	switch args[0] {
	case "init":
		return cmdInit(url)
	case "relay":
		if len(args) < 2 {
			return fmt.Errorf("usage: swarm-relay relay <local-file>")
		}
		return cmdRelay(url, args[1])
	case "stats":
		return cmdStats()
	case "reset":
		return cmdReset()
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func cmdInit(url string) error {
	if _, err := os.Stat(stateFile); err == nil {
		return fmt.Errorf("%s already exists — use `reset` first", stateFile)
	}
	batchHex := os.Getenv("BEE_BATCH_ID")
	if batchHex == "" {
		return fmt.Errorf("BEE_BATCH_ID is required")
	}
	batchID, err := swarm.BatchIDFromHex(batchHex)
	if err != nil {
		return fmt.Errorf("invalid BEE_BATCH_ID: %w", err)
	}
	signerHex := os.Getenv("BEE_SIGNER_HEX")
	if signerHex == "" {
		return fmt.Errorf("BEE_SIGNER_HEX is required")
	}
	if _, err := swarm.PrivateKeyFromHex(signerHex); err != nil {
		return fmt.Errorf("invalid BEE_SIGNER_HEX: %w", err)
	}

	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	batch, err := client.Postage.GetPostageBatch(context.Background(), batchID)
	if err != nil {
		return fmt.Errorf("get_postage_batch: %w", err)
	}

	st := &relayState{
		BatchID:   batchHex,
		SignerHex: signerHex,
		Depth:     batch.Depth,
		Buckets:   make([]uint32, numBuckets),
	}
	if err := save(st); err != nil {
		return err
	}
	fmt.Println("Relay initialised:")
	fmt.Printf("  batch:    %s\n", batch.BatchID.Hex())
	fmt.Printf("  depth:    %d\n", batch.Depth)
	fmt.Printf("  max_slot: %d\n", uint32(1)<<(batch.Depth-16))
	fmt.Println("\nReady. Use `swarm-relay relay <file>` to ingest.")
	return nil
}

func cmdRelay(url, path string) error {
	st, err := load()
	if err != nil {
		return err
	}
	batchID, err := swarm.BatchIDFromHex(st.BatchID)
	if err != nil {
		return fmt.Errorf("parse batch: %w", err)
	}
	signer, err := swarm.PrivateKeyFromHex(st.SignerHex)
	if err != nil {
		return fmt.Errorf("parse signer: %w", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	size := uint64(len(body))

	stamper, err := postage.NewStamper(signer, batchID, int(st.Depth))
	if err != nil {
		return fmt.Errorf("new_stamper: %w", err)
	}
	// Replay current buckets into the fresh stamper. bee-go's Stamper
	// has no public state mutator, so we re-stamp dummy addresses to
	// bring counters up. This is a starter-project shortcut; production
	// code would extend the Stamper to expose state hydration.
	if err := hydrateStamper(stamper, st.Buckets); err != nil {
		return fmt.Errorf("hydrate stamper: %w", err)
	}

	var stampErr error
	chunker := swarm.NewFileChunker(func(c swarm.Chunk) error {
		_, err := stamper.Stamp(c.Address.Raw())
		if err != nil {
			stampErr = err
			return err
		}
		st.Buckets[binBucket(c.Address.Raw())]++
		return nil
	})
	if _, err := chunker.Write(body); err != nil {
		st.RejectedFiles++
		_ = save(st)
		if stampErr != nil {
			return fmt.Errorf("rejected (likely bucket full): %w", stampErr)
		}
		return fmt.Errorf("chunker: %w", err)
	}
	root, err := chunker.Finalize()
	if err != nil {
		st.RejectedFiles++
		_ = save(st)
		if stampErr != nil {
			return fmt.Errorf("rejected (likely bucket full): %w", stampErr)
		}
		return fmt.Errorf("finalize: %w", err)
	}
	fmt.Printf("Pre-stamped %d bytes → root %s\n", size, root.Address.Hex())

	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	result, err := client.File.UploadData(context.Background(), batchID,
		bytes.NewReader(body), nil)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	if result.Reference.Hex() != root.Address.Hex() {
		fmt.Fprintf(os.Stderr, "warning: server-side ref %s differs from offline %s\n",
			result.Reference.Hex(), root.Address.Hex())
	}
	st.UploadedFiles++
	st.UploadedBytes += size
	if err := save(st); err != nil {
		return err
	}

	fmt.Printf("Uploaded → %s\n", result.Reference.Hex())
	fmt.Printf("  url: %s/bytes/%s\n", trim(url), result.Reference.Hex())
	printSummary(st)
	return nil
}

func cmdStats() error {
	st, err := load()
	if err != nil {
		return err
	}
	printSummary(st)
	maxSlot := uint64(1) << (st.Depth - 16)
	totalCapacity := maxSlot * uint64(numBuckets)
	var totalUsed uint64
	var maxHeight uint32
	hottest := 0
	for i, c := range st.Buckets {
		totalUsed += uint64(c)
		if c > maxHeight {
			maxHeight = c
			hottest = i
		}
	}
	fmt.Printf("Capacity:      %d / %d chunks\n", totalUsed, totalCapacity)
	fmt.Printf("Hottest bucket: #%04x (%d / %d)\n", hottest, maxHeight, maxSlot)
	return nil
}

func cmdReset() error {
	if _, err := os.Stat(stateFile); err == nil {
		if err := os.Remove(stateFile); err != nil {
			return fmt.Errorf("rm: %w", err)
		}
		fmt.Printf("Removed %s\n", stateFile)
	} else {
		fmt.Println("(nothing to reset)")
	}
	return nil
}

// hydrateStamper replays the persisted bucket counts into a fresh
// Stamper by stamping synthetic 32-byte addresses whose first 2 bytes
// land in each bucket. The signed envelopes are discarded — we only
// need the internal counters to match.
func hydrateStamper(s *postage.Stamper, buckets []uint32) error {
	addr := make([]byte, 32)
	for bucket, count := range buckets {
		if count == 0 {
			continue
		}
		addr[0] = byte(bucket >> 8)
		addr[1] = byte(bucket & 0xff)
		for i := uint32(0); i < count; i++ {
			if _, err := s.Stamp(addr); err != nil {
				return fmt.Errorf("hydrate bucket %04x: %w", bucket, err)
			}
		}
	}
	return nil
}

func binBucket(addr []byte) uint16 {
	return (uint16(addr[0]) << 8) | uint16(addr[1])
}

func printSummary(st *relayState) {
	fmt.Printf("Relay: depth=%d files_uploaded=%d bytes=%d files_rejected=%d\n",
		st.Depth, st.UploadedFiles, st.UploadedBytes, st.RejectedFiles)
}

func save(s *relayState) error {
	bytes, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, bytes, 0644)
}

func load() (*relayState, error) {
	bytes, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("%s not found — run `init` first", stateFile)
	}
	var s relayState
	if err := json.Unmarshal(bytes, &s); err != nil {
		return nil, err
	}
	if len(s.Buckets) != numBuckets {
		s.Buckets = make([]uint32, numBuckets)
	}
	return &s, nil
}

func trim(s string) string {
	if len(s) > 0 && s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}
	return s
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
