// Produce postage stamps offline using a client-side Stamper.
// Useful for batch-stamping pipelines, gas-free re-stamping, or
// delegating stamp generation to a service that holds the signing
// key. The bee-go Stamper is a fresh in-memory bucket counter — it
// does NOT persist state across runs (unlike bee-rs's Stamper, which
// exposes from_state for resumability).
//
// Usage:
//
//	go run ./examples/stamper-client-side
//
// Environment:
//   - BEE_URL        — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID   — usable postage batch (required)
//   - BEE_SIGNER_HEX — 32-byte hex private key (required). In
//     production this is the batch issuer's key.
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/postage"
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

	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	ctx := context.Background()

	batch, err := client.Postage.GetPostageBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("get_postage_batch: %w", err)
	}
	fmt.Println("Batch:")
	fmt.Printf("  id:    %s\n", batch.BatchID.Hex())
	fmt.Printf("  depth: %d\n", batch.Depth)
	fmt.Printf("  usable:%t\n\n", batch.Usable)

	stamper, err := postage.NewStamper(signer, batchID, int(batch.Depth))
	if err != nil {
		return fmt.Errorf("new_stamper: %w", err)
	}
	fmt.Println("Stamper:")
	fmt.Printf("  depth:    %d\n\n", batch.Depth)

	payloads := [][]byte{
		[]byte("alpha"),
		[]byte("beta"),
		[]byte("gamma"),
	}
	for _, p := range payloads {
		chunk, err := swarm.MakeContentAddressedChunk(p)
		if err != nil {
			return fmt.Errorf("make_cac: %w", err)
		}
		env, err := stamper.Stamp(chunk.Address.Raw())
		if err != nil {
			return fmt.Errorf("stamp: %w", err)
		}
		wire, err := postage.ConvertEnvelopeToMarshaledStamp(env)
		if err != nil {
			return fmt.Errorf("marshal_stamp: %w", err)
		}
		fmt.Printf("  payload %q\n", string(p))
		fmt.Printf("    chunk_addr: %s\n", chunk.Address.Hex())
		fmt.Printf("    issuer:     %s\n", env.Issuer.Hex())
		fmt.Printf("    index:      %s\n", hex.EncodeToString(env.Index))
		fmt.Printf("    timestamp:  %s\n", hex.EncodeToString(env.Timestamp))
		fmt.Printf("    stamp(%d): %s\n\n", postage.MarshaledStampLength, hex.EncodeToString(wire))
	}

	fmt.Println("Note: bee-go's Stamper is in-memory only — restart drops the bucket")
	fmt.Println("counters. For state persistence + resume across processes, see")
	fmt.Println("bee-rs's Stamper::from_state / state() snapshot API.")
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
