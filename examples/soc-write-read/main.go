// Sign and upload three Single Owner Chunks at distinct identifiers,
// then read them back via the SOC reader.
//
// A SOC's address is keccak256(identifier || owner). Anyone who knows
// (owner, identifier) can fetch and verify a chunk: the signature is
// recovered server-side and matched against the expected owner.
//
// Usage:
//
//	go run ./examples/soc-write-read
//
// Environment:
//   - BEE_URL        — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID   — usable postage batch (required)
//   - BEE_SIGNER_HEX — 32-byte hex private key (required)
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"unicode/utf8"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

type entry struct {
	label string
	body  []byte
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
	owner := signer.PublicKey().Address()

	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	ctx := context.Background()

	ecdsa, err := signer.ToECDSA()
	if err != nil {
		return fmt.Errorf("to_ecdsa: %w", err)
	}
	writer, err := client.File.MakeSOCWriter(ecdsa)
	if err != nil {
		return fmt.Errorf("make_soc_writer: %w", err)
	}
	fmt.Printf("SOC owner: %s\n", owner.Hex())

	entries := []entry{
		{label: "greeting", body: []byte("hello, single-owner-chunk")},
		{label: "number", body: []byte("42")},
		{label: "payload", body: []byte("\x01\x02\x03 binary bytes")},
	}

	fmt.Printf("\nWriting %d SOCs...\n", len(entries))
	for _, e := range entries {
		id := swarm.IdentifierFromString(e.label)
		result, err := writer.Upload(ctx, batchID, id, e.body, nil)
		if err != nil {
			return fmt.Errorf("upload %s: %w", e.label, err)
		}
		addr, err := swarm.CalculateSingleOwnerChunkAddress(id, owner)
		if err != nil {
			return fmt.Errorf("calc address %s: %w", e.label, err)
		}
		fmt.Printf("  %-8s id=%s  ref=%s  uploaded=%s\n",
			e.label, id.Hex(), addr.Hex(), result.Reference.Hex())
	}

	fmt.Println("\nReading back via SOC reader...")
	reader := client.File.MakeSOCReader(owner)
	for _, e := range entries {
		id := swarm.IdentifierFromString(e.label)
		soc, err := reader.Download(ctx, id)
		if err != nil {
			return fmt.Errorf("download %s: %w", e.label, err)
		}
		ownerOK := bytes.Equal(soc.Owner, owner.Raw())
		payloadOK := bytes.Equal(soc.Payload, e.body)
		if utf8.Valid(soc.Payload) {
			fmt.Printf("  %-8s payload=%q  owner_ok=%t  payload_ok=%t\n",
				e.label, string(soc.Payload), ownerOK, payloadOK)
		} else {
			fmt.Printf("  %-8s payload=(%d bytes binary)  owner_ok=%t  payload_ok=%t\n",
				e.label, len(soc.Payload), ownerOK, payloadOK)
		}
		if !ownerOK || !payloadOK {
			return fmt.Errorf("SOC verification failed for %s", e.label)
		}
	}

	fmt.Printf("\nAll %d SOCs verified.\n", len(entries))
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
