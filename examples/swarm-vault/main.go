// swarm-vault is a personal encrypted file dropbox.
//
// Files are encrypt-uploaded (Bee returns a 64-byte reference whose
// key travels inline). A name → encrypted_ref index is uploaded to
// Swarm and pointed at by a feed manifest, so the vault has one
// stable URL whose contents grow as files are added.
//
// The local .swarm-vault.json caches the current state for fast
// lookups; copy it (or just the feed manifest URL) to access the
// vault from another machine.
//
// Usage:
//
//	swarm-vault init  <name>
//	swarm-vault put   <name>  <local-file>
//	swarm-vault get   <name>  <out-path>
//	swarm-vault list
//	swarm-vault rm    <name>
//
// Environment:
//   - BEE_URL        — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID   — usable postage batch (required)
//   - BEE_SIGNER_HEX — 32-byte hex private key (required)
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

const stateFile = ".swarm-vault.json"

type indexEntry struct {
	EncryptedRef string `json:"encrypted_ref"`
	Size         int64  `json:"size"`
	TS           int64  `json:"ts"`
}

type vaultState struct {
	Name            string                `json:"name"`
	TopicHex        string                `json:"topic_hex"`
	OwnerHex        string                `json:"owner_hex"`
	FeedManifestRef string                `json:"feed_manifest_ref"`
	LatestIndexRef  string                `json:"latest_index_ref,omitempty"`
	Index           map[string]indexEntry `json:"index"`
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
		return fmt.Errorf("usage: swarm-vault <init|put|get|list|rm>")
	}
	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}

	switch args[0] {
	case "init":
		if len(args) < 2 {
			return fmt.Errorf("usage: swarm-vault init <name>")
		}
		return cmdInit(client, url, args[1])
	case "put":
		if len(args) < 3 {
			return fmt.Errorf("usage: swarm-vault put <name> <local-file>")
		}
		return cmdPut(client, args[1], args[2])
	case "get":
		if len(args) < 3 {
			return fmt.Errorf("usage: swarm-vault get <name> <out-path>")
		}
		return cmdGet(client, args[1], args[2])
	case "list":
		return cmdList()
	case "rm":
		if len(args) < 2 {
			return fmt.Errorf("usage: swarm-vault rm <name>")
		}
		return cmdRm(client, args[1])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func cmdInit(client *bee.Client, url, name string) error {
	if _, err := os.Stat(stateFile); err == nil {
		return fmt.Errorf("%s already exists", stateFile)
	}
	batchID, err := envBatch()
	if err != nil {
		return err
	}
	signer, err := envSigner()
	if err != nil {
		return err
	}
	owner := signer.PublicKey().Address()
	topic := swarm.TopicFromString("swarm-vault:" + name)

	feedManifest, err := client.File.CreateFeedManifest(context.Background(), batchID, owner, topic)
	if err != nil {
		return fmt.Errorf("create_feed_manifest: %w", err)
	}

	st := vaultState{
		Name:            name,
		TopicHex:        topic.Hex(),
		OwnerHex:        owner.Hex(),
		FeedManifestRef: feedManifest.Hex(),
		Index:           map[string]indexEntry{},
	}
	if err := save(&st); err != nil {
		return err
	}

	trimmed := strings.TrimRight(url, "/")
	fmt.Printf("Initialised vault %q\n", name)
	fmt.Printf("  feed manifest: %s\n", feedManifest.Hex())
	fmt.Printf("  vault URL:     %s/bzz/%s/\n", trimmed, feedManifest.Hex())
	fmt.Println("  (treat the URL as the vault password — anyone with it can read filenames.)")
	return nil
}

func cmdPut(client *bee.Client, name, local string) error {
	st, err := load()
	if err != nil {
		return err
	}
	batchID, err := envBatch()
	if err != nil {
		return err
	}
	signer, err := envSigner()
	if err != nil {
		return err
	}
	topic, err := swarm.TopicFromHex(st.TopicHex)
	if err != nil {
		return fmt.Errorf("parse topic: %w", err)
	}

	body, err := os.ReadFile(local)
	if err != nil {
		return fmt.Errorf("read %s: %w", local, err)
	}

	fmt.Printf("Uploading (encrypted) %d bytes...\n", len(body))
	yes := true
	opts := &api.RedundantUploadOptions{
		UploadOptions: api.UploadOptions{Encrypt: &yes},
	}
	// UploadData (POST /bytes) gives us a content-addressed leaf we
	// can download cleanly via /bytes again. UploadFile (POST /bzz)
	// would wrap the bytes in a single-fork manifest, requiring /bzz
	// to fetch and a filename to address.
	_ = name
	upload, err := client.File.UploadData(context.Background(), batchID,
		bytes.NewReader(body), opts)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	st.Index[name] = indexEntry{
		EncryptedRef: upload.Reference.Hex(),
		Size:         int64(len(body)),
		TS:           time.Now().Unix(),
	}
	if err := publishIndex(client, st, batchID, signer, topic); err != nil {
		return err
	}
	fmt.Printf("  vault entry: %s\n  encrypted_ref: %s\n", name, upload.Reference.Hex())
	return nil
}

func cmdGet(client *bee.Client, name, out string) error {
	st, err := load()
	if err != nil {
		return err
	}
	entry, ok := st.Index[name]
	if !ok {
		return fmt.Errorf("no such vault entry: %s", name)
	}
	ref, err := swarm.ReferenceFromHex(entry.EncryptedRef)
	if err != nil {
		return fmt.Errorf("parse ref: %w", err)
	}
	body, err := client.File.DownloadData(context.Background(), ref, nil)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer body.Close()
	got, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if err := os.WriteFile(out, got, 0644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	fmt.Printf("Wrote %d bytes to %s\n", len(got), out)
	return nil
}

func cmdList() error {
	st, err := load()
	if err != nil {
		return err
	}
	if len(st.Index) == 0 {
		fmt.Println("(empty vault)")
		return nil
	}
	fmt.Printf("vault %q  feed: %s\n", st.Name, st.FeedManifestRef)
	fmt.Printf("\n%-24s  %10s  %s\n", "name", "size", "encrypted_ref")
	keys := make([]string, 0, len(st.Index))
	for k := range st.Index {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		e := st.Index[k]
		fmt.Printf("%-24s  %10d  %s\n", k, e.Size, e.EncryptedRef)
	}
	return nil
}

func cmdRm(client *bee.Client, name string) error {
	st, err := load()
	if err != nil {
		return err
	}
	if _, ok := st.Index[name]; !ok {
		return fmt.Errorf("no such entry: %s", name)
	}
	delete(st.Index, name)
	batchID, err := envBatch()
	if err != nil {
		return err
	}
	signer, err := envSigner()
	if err != nil {
		return err
	}
	topic, err := swarm.TopicFromHex(st.TopicHex)
	if err != nil {
		return fmt.Errorf("parse topic: %w", err)
	}
	if err := publishIndex(client, st, batchID, signer, topic); err != nil {
		return err
	}
	fmt.Printf("Removed %s\n", name)
	return nil
}

func publishIndex(client *bee.Client, st *vaultState, batchID swarm.BatchID,
	signer swarm.PrivateKey, topic swarm.Topic) error {
	data, err := json.MarshalIndent(st.Index, "", "  ")
	if err != nil {
		return err
	}
	r, err := client.File.UploadData(context.Background(), batchID, bytes.NewReader(data), nil)
	if err != nil {
		return fmt.Errorf("upload index: %w", err)
	}
	if _, err := client.File.UpdateFeedWithReference(context.Background(),
		batchID, signer, topic, r.Reference, nil); err != nil {
		return fmt.Errorf("update_feed: %w", err)
	}
	st.LatestIndexRef = r.Reference.Hex()
	return save(st)
}

func envBatch() (swarm.BatchID, error) {
	h := os.Getenv("BEE_BATCH_ID")
	if h == "" {
		return swarm.BatchID{}, fmt.Errorf("BEE_BATCH_ID is required")
	}
	return swarm.BatchIDFromHex(h)
}

func envSigner() (swarm.PrivateKey, error) {
	h := os.Getenv("BEE_SIGNER_HEX")
	if h == "" {
		return swarm.PrivateKey{}, fmt.Errorf("BEE_SIGNER_HEX is required")
	}
	return swarm.PrivateKeyFromHex(h)
}

func save(s *vaultState) error {
	bytes, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, bytes, 0644)
}

func load() (*vaultState, error) {
	bytes, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("%s not found — run `init` first", stateFile)
	}
	var s vaultState
	if err := json.Unmarshal(bytes, &s); err != nil {
		return nil, err
	}
	if s.Index == nil {
		s.Index = map[string]indexEntry{}
	}
	return &s, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
