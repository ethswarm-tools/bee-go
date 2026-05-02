// Upload a file under an Access Control Trie (ACT), manage the
// grantee list (create / get / patch), and download as the publisher
// using the resulting history root. ACT lets the owner of an upload
// share encrypted content with a set of public keys without
// re-uploading.
//
// Usage:
//
//	go run ./examples/act-share
//
// Environment:
//   - BEE_URL      — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID — usable postage batch (required)
//
// The publisher is the local Bee node — Bee signs ACT operations with
// the node's identity and uses its publicKey from GET /addresses for
// download authorisation.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/api"
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
	// Bee's /addresses returns the public key as 33-byte compressed
	// SEC1 hex. swarm.NewPublicKey decompresses it to the 64-byte
	// uncompressed form ACT calls expect.
	publisherPKBytes, err := hex.DecodeString(addresses.PublicKey)
	if err != nil {
		return fmt.Errorf("decode publisher pubkey hex: %w", err)
	}
	publisherPK, err := swarm.NewPublicKey(publisherPKBytes)
	if err != nil {
		return fmt.Errorf("parse publisher pubkey: %w", err)
	}
	fmt.Printf("Publisher (node) pubkey: %s\n", addresses.PublicKey)

	g1, err := randomCompressedPubkey()
	if err != nil {
		return err
	}
	g2, err := randomCompressedPubkey()
	if err != nil {
		return err
	}
	g3, err := randomCompressedPubkey()
	if err != nil {
		return err
	}
	fmt.Printf("Generated grantees:\n  %s\n  %s\n  %s\n\n", g1, g2, g3)

	yes := true
	payload := []byte("hello act grantees!")
	uploadOpts := &api.FileUploadOptions{
		UploadOptions: api.UploadOptions{Act: &yes},
		ContentType:   "text/plain",
	}
	upload, err := client.File.UploadFile(ctx, batchID,
		bytes.NewReader(payload), "act-secret.txt", "text/plain", uploadOpts)
	if err != nil {
		return fmt.Errorf("upload_file: %w", err)
	}
	if upload.HistoryAddress == nil {
		return fmt.Errorf("upload did not return ACT history address")
	}
	history := *upload.HistoryAddress
	fmt.Println("Uploaded:")
	fmt.Printf("  reference:       %s\n", upload.Reference.Hex())
	fmt.Printf("  history_address: %s\n\n", history.Hex())

	created, err := client.API.CreateGrantees(ctx, batchID, []string{g1, g2, g3})
	if err != nil {
		return fmt.Errorf("create_grantees: %w", err)
	}
	createdRef, err := swarm.ReferenceFromHex(created.Ref)
	if err != nil {
		return fmt.Errorf("parse created ref: %w", err)
	}
	fmt.Println("Created grantee list:")
	fmt.Printf("  ref:       %s\n", created.Ref)
	fmt.Printf("  historyref:%s\n", created.HistoryRef)
	listed, err := client.API.GetGrantees(ctx, createdRef)
	if err != nil {
		return fmt.Errorf("get_grantees: %w", err)
	}
	fmt.Printf("  members (%d): %v\n\n", len(listed), listed)

	// Bee needs a moment to settle the grantee chunk after
	// CreateGrantees; bee-js's integration test waits 5s, 2s suffices
	// for a local node.
	time.Sleep(2 * time.Second)
	patched, err := client.API.PatchGrantees(ctx, batchID, createdRef, history,
		[]string{g1}, []string{g2, g3})
	if err != nil {
		return fmt.Errorf("patch_grantees: %w", err)
	}
	patchedRef, err := swarm.ReferenceFromHex(patched.Ref)
	if err != nil {
		return fmt.Errorf("parse patched ref: %w", err)
	}
	patchedAfter, err := client.API.GetGrantees(ctx, patchedRef)
	if err != nil {
		return fmt.Errorf("get_grantees after patch: %w", err)
	}
	fmt.Println("After patch (add 1, revoke 2):")
	fmt.Printf("  ref:       %s\n", patched.Ref)
	fmt.Printf("  historyref:%s\n", patched.HistoryRef)
	fmt.Printf("  members (%d): %v\n\n", len(patchedAfter), patchedAfter)

	downloadOpts := &api.DownloadOptions{
		ActPublisher:      &publisherPK,
		ActHistoryAddress: &history,
		ActTimestamp:      time.Now().Unix(),
	}
	// /bzz resolves the manifest down to the leaf file content (and
	// decrypts under ACT). /bytes would return the raw manifest chunk
	// bytes here, not the original payload.
	body, _, err := client.File.DownloadFile(ctx, upload.Reference, downloadOpts)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer body.Close()
	got, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("read download: %w", err)
	}
	fmt.Printf("Downloaded %d bytes\n", len(got))
	fmt.Printf("  payload: %q\n", string(got))
	if !bytes.Equal(got, payload) {
		return fmt.Errorf("ACT round-trip payload mismatch")
	}
	fmt.Println("\nRound-trip OK: ACT-protected upload decrypted via publisher identity.")
	return nil
}

func randomCompressedPubkey() (string, error) {
	var seed [32]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return "", err
	}
	pk, err := swarm.NewPrivateKey(seed[:])
	if err != nil {
		return "", err
	}
	return pk.PublicKey().CompressedHex()
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
