// Demonstrate Generic SOC pub/sub:
//
//  1. Read the local node's overlay address.
//  2. Mine a signer (PoW-style) so the SOC address
//     keccak256(identifier || signer.address) lands in that overlay's
//     neighbourhood.
//  3. Open a websocket subscription on that SOC address.
//  4. Send three GSOC messages with the mined signer.
//  5. Receive them on the subscription side.
//
// Usage:
//
//	go run ./examples/gsoc-mined-pubsub
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
	"time"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/crypto"

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

	addresses, err := client.Debug.Addresses(ctx)
	if err != nil {
		return fmt.Errorf("addresses: %w", err)
	}
	overlayBytes, err := hex.DecodeString(addresses.Overlay)
	if err != nil {
		return fmt.Errorf("invalid overlay hex: %w", err)
	}
	fmt.Printf("Node overlay: %s\n", addresses.Overlay)

	identifier := swarm.IdentifierFromString("demo-channel")
	proximity := 8
	fmt.Printf("Mining GSOC signer for identifier %s at proximity %d...\n",
		identifier.Hex(), proximity)
	ecdsaKey, err := swarm.GSOCMine(overlayBytes, identifier.Raw(), proximity)
	if err != nil {
		return fmt.Errorf("gsoc_mine: %w", err)
	}
	signer, err := swarm.NewPrivateKey(crypto.FromECDSA(ecdsaKey))
	if err != nil {
		return fmt.Errorf("convert mined signer: %w", err)
	}
	owner := signer.PublicKey().Address()
	socAddr, err := swarm.CalculateSingleOwnerChunkAddress(identifier, owner)
	if err != nil {
		return fmt.Errorf("soc_address: %w", err)
	}
	fmt.Printf("  signer.address: %s\n", owner.Hex())
	fmt.Printf("  soc_address:    %s\n\n", socAddr.Hex())

	sub, err := client.GSOC.Subscribe(ctx, owner, identifier)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}
	defer sub.Cancel()

	// Each message is uniquely tagged so we can recognise our own
	// sends and ignore any prior-run replays Bee may push us right
	// after subscribe.
	nonce := time.Now().UnixNano()
	messages := [][]byte{
		fmt.Appendf(nil, "%d:hello gsoc", nonce),
		fmt.Appendf(nil, "%d:second message", nonce),
		fmt.Appendf(nil, "%d:third and last", nonce),
	}

	go func() {
		// Give the websocket a moment to register and absorb any
		// replay before we start writing.
		time.Sleep(2 * time.Second)
		for i, msg := range messages {
			if _, err := client.GSOC.Send(ctx, batchID, signer, identifier, msg, nil); err != nil {
				fmt.Fprintf(os.Stderr, "send #%d failed: %v\n", i, err)
				return
			}
			fmt.Printf("  -> sent #%d: %d bytes\n", i, len(msg))
			// Each send goes to the same SOC address (same
			// identifier + signer), so a new put overwrites the
			// previous one in Bee's local store. Spacing the writes
			// gives Bee time to fire its websocket notification
			// before the next put — bursts get coalesced.
			time.Sleep(3 * time.Second)
		}
	}()

	// We accept the messages in any order and ignore unknown payloads
	// (replayed chunks from a prior run, or coalesced bursts).
	expected := map[string]bool{}
	for _, m := range messages {
		expected[string(m)] = true
	}
	fmt.Printf("Listening for %d messages...\n", len(messages))
	deadline := time.After(45 * time.Second)
	for len(expected) > 0 {
		select {
		case msg, ok := <-sub.Messages:
			if !ok {
				return fmt.Errorf("subscription closed early")
			}
			key := string(msg)
			if expected[key] {
				delete(expected, key)
				if utf8.Valid(msg) {
					fmt.Printf("  <- recv: %q\n", string(msg))
				} else {
					fmt.Printf("  <- recv: (%d bytes binary)\n", len(msg))
				}
			} else {
				fmt.Printf("  (ignored unexpected chunk: %d bytes)\n", len(msg))
			}
		case err := <-sub.Errors:
			return fmt.Errorf("subscription error: %w", err)
		case <-deadline:
			return fmt.Errorf("timeout waiting for messages: %d still expected", len(expected))
		}
	}

	fmt.Printf("\nGSOC round-trip OK: %d messages received.\n", len(messages))
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
