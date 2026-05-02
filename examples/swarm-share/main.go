// swarm-share is revocable file sharing on Swarm.
//
// `share <file> --to <pubkey>...` uploads the file under an Access
// Control Trie (ACT), creates a grantee list with the provided
// recipient public keys, and prints the references the recipients
// need to download. `revoke` patches the grantee list to drop a key
// without re-uploading the file.
//
// Usage:
//
//	swarm-share share  <file>  --to <pubkey>...
//	swarm-share list
//	swarm-share revoke <id>    --grantee <pubkey>
//	swarm-share grantees <id>
//
// Environment:
//   - BEE_URL      — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID — usable postage batch (required)
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

const stateFile = ".swarm-share.json"

type share struct {
	ID             string   `json:"id"`
	File           string   `json:"file"`
	FileRef        string   `json:"file_ref"`
	HistoryAddress string   `json:"history_address"`
	GranteeRef     string   `json:"grantee_ref"`
	GranteeHistory string   `json:"grantee_history"`
	Grantees       []string `json:"grantees"`
	TS             int64    `json:"ts"`
}

type shares struct {
	Shares []share `json:"shares"`
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
		return fmt.Errorf("usage: swarm-share <share|list|revoke|grantees>")
	}
	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	switch args[0] {
	case "share":
		return cmdShare(client, args[1:])
	case "list":
		return cmdList()
	case "revoke":
		return cmdRevoke(client, args[1:])
	case "grantees":
		return cmdGrantees(client, args[1:])
	default:
		_ = url
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func cmdShare(client *bee.Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: swarm-share share <file> --to <pubkey>")
	}
	file := args[0]
	var grantees []string
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--to":
			i++
			if i >= len(args) {
				return fmt.Errorf("--to needs a value")
			}
			grantees = append(grantees, args[i])
		default:
			return fmt.Errorf("unknown flag: %s", args[i])
		}
	}
	if len(grantees) == 0 {
		return fmt.Errorf("--to <pubkey> required at least once")
	}
	batchID, err := envBatch()
	if err != nil {
		return err
	}

	body, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read %s: %w", file, err)
	}
	name := filepath.Base(file)
	yes := true

	fmt.Printf("Uploading %s under ACT...\n", file)
	opts := &api.FileUploadOptions{
		UploadOptions: api.UploadOptions{Act: &yes},
		ContentType:   "application/octet-stream",
	}
	upload, err := client.File.UploadFile(context.Background(), batchID,
		bytes.NewReader(body), name, "application/octet-stream", opts)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	if upload.HistoryAddress == nil {
		return fmt.Errorf("upload did not return ACT history address")
	}
	history := *upload.HistoryAddress
	fmt.Printf("  file ref:        %s\n", upload.Reference.Hex())
	fmt.Printf("  history_address: %s\n", history.Hex())

	fmt.Printf("Creating grantee list (%d keys)...\n", len(grantees))
	created, err := client.API.CreateGrantees(context.Background(), batchID, grantees)
	if err != nil {
		return fmt.Errorf("create_grantees: %w", err)
	}
	fmt.Printf("  grantee ref:     %s\n", created.Ref)
	fmt.Printf("  grantee history: %s\n", created.HistoryRef)

	id := fmt.Sprintf("%08x", uint32(time.Now().Unix()))
	st := load()
	st.Shares = append(st.Shares, share{
		ID:             id,
		File:           name,
		FileRef:        upload.Reference.Hex(),
		HistoryAddress: history.Hex(),
		GranteeRef:     created.Ref,
		GranteeHistory: created.HistoryRef,
		Grantees:       grantees,
		TS:             time.Now().Unix(),
	})
	if err := save(st); err != nil {
		return err
	}

	fmt.Printf("\nRecipient instructions for share %s:\n", id)
	fmt.Println("  set BEE_URL to a node where the recipient is the publisher,")
	fmt.Println("  then download with these headers:")
	fmt.Println("    Swarm-Act:                 true")
	fmt.Println("    Swarm-Act-Publisher:       <publisher's compressed pubkey>")
	fmt.Printf("    Swarm-Act-History-Address: %s\n", history.Hex())
	fmt.Println("    Swarm-Act-Timestamp:       <current unix time>")
	fmt.Printf("  on /bzz/%s/\n", upload.Reference.Hex())
	return nil
}

func cmdList() error {
	st := load()
	if len(st.Shares) == 0 {
		fmt.Println("(no shares yet)")
		return nil
	}
	fmt.Printf("%-10s  %-20s  %-10s  %s\n", "id", "file", "grantees", "file_ref")
	for _, s := range st.Shares {
		fmt.Printf("%-10s  %-20s  %-10d  %s\n", s.ID, truncate(s.File, 20), len(s.Grantees), s.FileRef)
	}
	return nil
}

func cmdRevoke(client *bee.Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: swarm-share revoke <id> --grantee <pubkey>")
	}
	id := args[0]
	var toRevoke []string
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--grantee":
			i++
			if i >= len(args) {
				return fmt.Errorf("--grantee needs a value")
			}
			toRevoke = append(toRevoke, args[i])
		default:
			return fmt.Errorf("unknown flag: %s", args[i])
		}
	}
	if len(toRevoke) == 0 {
		return fmt.Errorf("--grantee <pubkey> required")
	}

	st := load()
	pos := -1
	for i, s := range st.Shares {
		if s.ID == id {
			pos = i
			break
		}
	}
	if pos < 0 {
		return fmt.Errorf("no share with id %s", id)
	}
	batchID, err := envBatch()
	if err != nil {
		return err
	}
	s := &st.Shares[pos]
	granteeRef, err := swarm.ReferenceFromHex(s.GranteeRef)
	if err != nil {
		return fmt.Errorf("parse grantee ref: %w", err)
	}
	history, err := swarm.ReferenceFromHex(s.HistoryAddress)
	if err != nil {
		return fmt.Errorf("parse history: %w", err)
	}
	patched, err := client.API.PatchGrantees(context.Background(), batchID,
		granteeRef, history, nil, toRevoke)
	if err != nil {
		return fmt.Errorf("patch_grantees: %w", err)
	}
	s.GranteeRef = patched.Ref
	s.GranteeHistory = patched.HistoryRef
	revoked := map[string]bool{}
	for _, r := range toRevoke {
		revoked[r] = true
	}
	keep := s.Grantees[:0]
	for _, g := range s.Grantees {
		if !revoked[g] {
			keep = append(keep, g)
		}
	}
	s.Grantees = keep
	if err := save(st); err != nil {
		return err
	}
	fmt.Printf("Revoked %d grantee(s) from share %s\n", len(toRevoke), id)
	fmt.Printf("  new grantee ref: %s\n", patched.Ref)
	return nil
}

func cmdGrantees(client *bee.Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: swarm-share grantees <id>")
	}
	id := args[0]
	st := load()
	var s *share
	for i := range st.Shares {
		if st.Shares[i].ID == id {
			s = &st.Shares[i]
			break
		}
	}
	if s == nil {
		return fmt.Errorf("no share with id %s", id)
	}
	r, err := swarm.ReferenceFromHex(s.GranteeRef)
	if err != nil {
		return fmt.Errorf("parse grantee ref: %w", err)
	}
	live, err := client.API.GetGrantees(context.Background(), r)
	if err != nil {
		return fmt.Errorf("get_grantees: %w", err)
	}
	fmt.Printf("share %s: %s\n", id, s.File)
	fmt.Printf("  cached: %d grantees\n", len(s.Grantees))
	for _, g := range s.Grantees {
		fmt.Printf("    %s\n", g)
	}
	fmt.Printf("  live:   %d grantees\n", len(live))
	for _, g := range live {
		fmt.Printf("    %s\n", g)
	}
	return nil
}

func envBatch() (swarm.BatchID, error) {
	h := os.Getenv("BEE_BATCH_ID")
	if h == "" {
		return swarm.BatchID{}, fmt.Errorf("BEE_BATCH_ID is required")
	}
	return swarm.BatchIDFromHex(h)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func load() *shares {
	bytes, err := os.ReadFile(stateFile)
	if err != nil {
		return &shares{}
	}
	var s shares
	if err := json.Unmarshal(bytes, &s); err != nil {
		return &shares{}
	}
	return &s
}

func save(s *shares) error {
	bytes, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, bytes, 0644)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
