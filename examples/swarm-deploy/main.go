// swarm-deploy is a `git push`-style site deploy tool on Swarm.
//
// Each project gets a feed manifest (one stable /bzz/<feedRef>/ URL);
// each `push` re-uploads the directory, updates the feed pointer,
// and appends a row to .swarmdeploy/state.json for history. Rollback
// rewinds the feed to a past upload's reference.
//
// Usage:
//
//	swarm-deploy init  <topic-name>            # one-time, creates feed manifest
//	swarm-deploy push  <local-dir> [note]      # upload + update feed
//	swarm-deploy history                       # list past versions
//	swarm-deploy rollback <index>              # point feed back at version <index>
//
// Environment:
//   - BEE_URL        — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID   — usable postage batch (required)
//   - BEE_SIGNER_HEX — 32-byte hex private key (required)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

const statePath = ".swarmdeploy/state.json"

type historyEntry struct {
	Timestamp int64  `json:"timestamp"`
	SiteRef   string `json:"site_ref"`
	Note      string `json:"note"`
}

type state struct {
	TopicName       string         `json:"topic_name"`
	TopicHex        string         `json:"topic_hex"`
	OwnerHex        string         `json:"owner_hex"`
	FeedManifestRef string         `json:"feed_manifest_ref"`
	History         []historyEntry `json:"history"`
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
		return fmt.Errorf("usage: swarm-deploy <init|push|history|rollback>")
	}
	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}

	switch args[0] {
	case "init":
		if len(args) < 2 {
			return fmt.Errorf("usage: swarm-deploy init <topic-name>")
		}
		return cmdInit(client, url, args[1])
	case "push":
		if len(args) < 2 {
			return fmt.Errorf("usage: swarm-deploy push <local-dir> [note]")
		}
		note := ""
		if len(args) >= 3 {
			note = args[2]
		}
		return cmdPush(client, url, args[1], note)
	case "history":
		return cmdHistory()
	case "rollback":
		if len(args) < 2 {
			return fmt.Errorf("usage: swarm-deploy rollback <index>")
		}
		idx, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid index: %w", err)
		}
		return cmdRollback(client, url, idx)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func cmdInit(client *bee.Client, url, topicName string) error {
	if _, err := os.Stat(statePath); err == nil {
		return fmt.Errorf("%s already exists — already initialised", statePath)
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
	topic := swarm.TopicFromString(topicName)

	feedManifest, err := client.File.CreateFeedManifest(context.Background(), batchID, owner, topic)
	if err != nil {
		return fmt.Errorf("create_feed_manifest: %w", err)
	}

	st := state{
		TopicName:       topicName,
		TopicHex:        topic.Hex(),
		OwnerHex:        owner.Hex(),
		FeedManifestRef: feedManifest.Hex(),
	}
	if err := saveState(&st); err != nil {
		return err
	}

	trimmed := strings.TrimRight(url, "/")
	fmt.Printf("Initialised swarm-deploy for %q\n", topicName)
	fmt.Printf("  feed manifest: %s\n", feedManifest.Hex())
	fmt.Printf("  stable URL:    %s/bzz/%s/\n", trimmed, feedManifest.Hex())
	fmt.Println("\n(Empty until first `swarm-deploy push <dir>`.)")
	return nil
}

func cmdPush(client *bee.Client, url, dir, note string) error {
	st, err := loadState()
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

	fmt.Printf("Uploading %s...\n", dir)
	result, err := client.File.UploadCollection(context.Background(), batchID, dir, nil)
	if err != nil {
		return fmt.Errorf("upload_collection: %w", err)
	}
	siteRef := result.Reference
	fmt.Printf("  site ref: %s\n", siteRef.Hex())

	fmt.Println("Updating feed pointer...")
	if _, err := client.File.UpdateFeedWithReference(context.Background(),
		batchID, signer, topic, siteRef, nil); err != nil {
		return fmt.Errorf("update_feed: %w", err)
	}

	st.History = append(st.History, historyEntry{
		Timestamp: time.Now().Unix(),
		SiteRef:   siteRef.Hex(),
		Note:      note,
	})
	if err := saveState(st); err != nil {
		return err
	}

	trimmed := strings.TrimRight(url, "/")
	fmt.Printf("\nDeployed v%d\n", len(st.History))
	fmt.Printf("  stable URL: %s/bzz/%s/\n", trimmed, st.FeedManifestRef)
	fmt.Printf("  this rev:   %s/bzz/%s/\n", trimmed, siteRef.Hex())
	return nil
}

func cmdHistory() error {
	st, err := loadState()
	if err != nil {
		return err
	}
	if len(st.History) == 0 {
		fmt.Println("(no deploys yet)")
		return nil
	}
	fmt.Printf("topic:         %s\n", st.TopicName)
	fmt.Printf("feed manifest: %s\n\n", st.FeedManifestRef)
	fmt.Printf("%-3s %-10s  %-64s  note\n", "#", "ts", "site_ref")
	for i, e := range st.History {
		fmt.Printf("%-3d %-10d  %-64s  %s\n", i, e.Timestamp, e.SiteRef, e.Note)
	}
	return nil
}

func cmdRollback(client *bee.Client, url string, idx int) error {
	st, err := loadState()
	if err != nil {
		return err
	}
	if idx < 0 || idx >= len(st.History) {
		return fmt.Errorf("no version at index %d", idx)
	}
	target := st.History[idx]
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
	siteRef, err := swarm.ReferenceFromHex(target.SiteRef)
	if err != nil {
		return fmt.Errorf("parse site_ref: %w", err)
	}

	if _, err := client.File.UpdateFeedWithReference(context.Background(),
		batchID, signer, topic, siteRef, nil); err != nil {
		return fmt.Errorf("update_feed: %w", err)
	}
	st.History = append(st.History, historyEntry{
		Timestamp: time.Now().Unix(),
		SiteRef:   target.SiteRef,
		Note:      fmt.Sprintf("rollback to #%d", idx),
	})
	if err := saveState(st); err != nil {
		return err
	}

	trimmed := strings.TrimRight(url, "/")
	fmt.Printf("Rolled back to #%d: %s\n", idx, target.SiteRef)
	fmt.Printf("  stable URL: %s/bzz/%s/\n", trimmed, st.FeedManifestRef)
	return nil
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

func saveState(s *state) error {
	if err := os.MkdirAll(filepath.Dir(statePath), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	bytes, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath, bytes, 0644)
}

func loadState() (*state, error) {
	bytes, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("%s not found — run `init` first", statePath)
	}
	var s state
	if err := json.Unmarshal(bytes, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
