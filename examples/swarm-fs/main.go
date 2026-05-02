// swarm-fs is filesystem-style staging for a Mantaray collection.
//
// Maintain a local "staging tree" in .swarm-fs.json mapping
// <path-in-manifest> → <local-file-on-disk>. Mutate it like a
// filesystem (add, mv, rm, ls); when you're happy, `publish` reads
// each local file, packages them as a Mantaray collection, and prints
// the resulting Swarm reference. The staging file is the single
// source of truth — bee-rs/bee-go don't yet expose HTTP-aware
// load_recursively for in-place manifest mutation, so we round-trip
// through local state instead.
//
// Usage:
//
//	swarm-fs init
//	swarm-fs add  <path-in-manifest>  <local-file>
//	swarm-fs mv   <old-path>          <new-path>
//	swarm-fs rm   <path>
//	swarm-fs ls
//	swarm-fs publish [--index <name>]
//
// Environment:
//   - BEE_URL      — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID — usable postage batch (required for publish)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/file"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

const statePath = ".swarm-fs.json"

type tree struct {
	Entries map[string]string `json:"entries"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]
	if len(args) == 0 {
		return fmt.Errorf("usage: swarm-fs <init|add|mv|rm|ls|publish>")
	}
	switch args[0] {
	case "init":
		return cmdInit()
	case "add":
		if len(args) < 3 {
			return fmt.Errorf("usage: swarm-fs add <path> <local-file>")
		}
		return cmdAdd(args[1], args[2])
	case "mv":
		if len(args) < 3 {
			return fmt.Errorf("usage: swarm-fs mv <old> <new>")
		}
		return cmdMv(args[1], args[2])
	case "rm":
		if len(args) < 2 {
			return fmt.Errorf("usage: swarm-fs rm <path>")
		}
		return cmdRm(args[1])
	case "ls":
		return cmdLs()
	case "publish":
		return cmdPublish(args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func cmdInit() error {
	if _, err := os.Stat(statePath); err == nil {
		return fmt.Errorf("%s already exists", statePath)
	}
	return save(&tree{Entries: map[string]string{}})
}

func cmdAdd(path, local string) error {
	p, err := normalise(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(local)
	if err != nil || info.IsDir() {
		return fmt.Errorf("%s is not a regular file", local)
	}
	t, err := load()
	if err != nil {
		return err
	}
	t.Entries[p] = local
	if err := save(t); err != nil {
		return err
	}
	fmt.Printf("staged %s → %s\n", p, local)
	return nil
}

func cmdMv(old, new string) error {
	op, err := normalise(old)
	if err != nil {
		return err
	}
	np, err := normalise(new)
	if err != nil {
		return err
	}
	t, err := load()
	if err != nil {
		return err
	}
	v, ok := t.Entries[op]
	if !ok {
		return fmt.Errorf("no such path: %s", op)
	}
	delete(t.Entries, op)
	t.Entries[np] = v
	if err := save(t); err != nil {
		return err
	}
	fmt.Printf("renamed %s → %s\n", op, np)
	return nil
}

func cmdRm(path string) error {
	p, err := normalise(path)
	if err != nil {
		return err
	}
	t, err := load()
	if err != nil {
		return err
	}
	if _, ok := t.Entries[p]; !ok {
		return fmt.Errorf("no such path: %s", p)
	}
	delete(t.Entries, p)
	if err := save(t); err != nil {
		return err
	}
	fmt.Printf("removed %s\n", p)
	return nil
}

func cmdLs() error {
	t, err := load()
	if err != nil {
		return err
	}
	if len(t.Entries) == 0 {
		fmt.Println("(empty)")
		return nil
	}
	keys := make([]string, 0, len(t.Entries))
	for k := range t.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Printf("%-40s  %s\n", "path", "local-file")
	for _, k := range keys {
		fmt.Printf("%-40s  %s\n", k, t.Entries[k])
	}
	return nil
}

func cmdPublish(extra []string) error {
	url := getenv("BEE_URL", "http://localhost:1633")
	batchHex := os.Getenv("BEE_BATCH_ID")
	if batchHex == "" {
		return fmt.Errorf("BEE_BATCH_ID is required")
	}
	batchID, err := swarm.BatchIDFromHex(batchHex)
	if err != nil {
		return fmt.Errorf("invalid BEE_BATCH_ID: %w", err)
	}

	t, err := load()
	if err != nil {
		return err
	}
	if len(t.Entries) == 0 {
		return fmt.Errorf("nothing to publish")
	}

	indexDoc := ""
	for i := 0; i < len(extra); i++ {
		switch extra[i] {
		case "--index":
			i++
			if i >= len(extra) {
				return fmt.Errorf("--index needs a value")
			}
			indexDoc = extra[i]
		default:
			return fmt.Errorf("unknown publish flag: %s", extra[i])
		}
	}

	keys := make([]string, 0, len(t.Entries))
	for k := range t.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var entries []file.CollectionEntry
	for _, k := range keys {
		data, err := os.ReadFile(t.Entries[k])
		if err != nil {
			return fmt.Errorf("read %s: %w", t.Entries[k], err)
		}
		entries = append(entries, file.CollectionEntry{Path: k, Data: data})
	}

	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	opts := &api.CollectionUploadOptions{IndexDocument: indexDoc}
	result, err := client.File.UploadCollectionEntries(context.Background(), batchID, entries, opts)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	trimmed := strings.TrimRight(url, "/")
	fmt.Printf("Uploaded %d entries.\n", len(entries))
	fmt.Printf("  reference: %s\n", result.Reference.Hex())
	fmt.Printf("  url:       %s/bzz/%s/\n", trimmed, result.Reference.Hex())
	if indexDoc != "" {
		fmt.Printf("  index_document: %s\n", indexDoc)
	}
	return nil
}

func normalise(p string) (string, error) {
	p = strings.TrimSpace(strings.TrimLeft(p, "/"))
	if p == "" {
		return "", fmt.Errorf("empty path")
	}
	return p, nil
}

func save(t *tree) error {
	bytes, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath, bytes, 0644)
}

func load() (*tree, error) {
	bytes, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("%s not found — run `init` first", statePath)
	}
	var t tree
	if err := json.Unmarshal(bytes, &t); err != nil {
		return nil, err
	}
	if t.Entries == nil {
		t.Entries = map[string]string{}
	}
	return &t, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
