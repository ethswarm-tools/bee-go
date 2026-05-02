// swarm-pinner watches a directory, uploads+pins new files, and
// periodically checks that pinned content is still retrievable.
//
// Polls every --interval seconds (default 5). Each new regular file
// under <watch-dir> is uploaded with pin: true; existing pinned items
// are re-checked with IsRetrievable. State persists in
// .swarm-pinner.json so restarts don't re-upload.
//
// Usage:
//
//	swarm-pinner <watch-dir>          # run forever
//	swarm-pinner <watch-dir> --once   # one pass + exit
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
	"sort"
	"strconv"
	"time"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

const stateFile = ".swarm-pinner.json"

type pinnedEntry struct {
	Reference   string `json:"reference"`
	Size        int64  `json:"size"`
	PinnedAt    int64  `json:"pinned_at"`
	LastCheckOK bool   `json:"last_check_ok"`
	LastCheckAt int64  `json:"last_check_at"`
}

type state struct {
	Pinned map[string]pinnedEntry `json:"pinned"`
}

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

	args := os.Args[1:]
	if len(args) == 0 {
		return fmt.Errorf("usage: swarm-pinner <watch-dir> [--once] [--interval N]")
	}
	dir := args[0]
	once := false
	intervalSecs := 5
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--once":
			once = true
		case "--interval":
			i++
			if i >= len(args) {
				return fmt.Errorf("--interval needs N")
			}
			n, err := strconv.Atoi(args[i])
			if err != nil {
				return fmt.Errorf("invalid interval: %w", err)
			}
			intervalSecs = n
		default:
			return fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	fmt.Printf("Watching %s (every %ds)\n", dir, intervalSecs)

	for {
		if err := pass(client, batchID, dir); err != nil {
			fmt.Fprintf(os.Stderr, "pass error: %v\n", err)
		}
		if once {
			break
		}
		time.Sleep(time.Duration(intervalSecs) * time.Second)
	}
	return nil
}

func pass(client *bee.Client, batchID swarm.BatchID, dir string) error {
	st := load()
	now := time.Now().Unix()

	files, err := listFiles(dir)
	if err != nil {
		return err
	}
	for _, path := range files {
		if _, ok := st.Pinned[path]; ok {
			continue
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		size := int64(len(body))
		name := filepath.Base(path)
		yes := true
		opts := &api.FileUploadOptions{
			UploadOptions: api.UploadOptions{Pin: &yes},
			ContentType:   "application/octet-stream",
		}
		result, err := client.File.UploadFile(context.Background(), batchID,
			bytes.NewReader(body), name, "application/octet-stream", opts)
		if err != nil {
			return fmt.Errorf("upload %s: %w", path, err)
		}
		fmt.Printf("[%d] uploaded+pinned %s (%d bytes) → %s\n",
			now, path, size, result.Reference.Hex())
		st.Pinned[path] = pinnedEntry{
			Reference:   result.Reference.Hex(),
			Size:        size,
			PinnedAt:    now,
			LastCheckOK: true,
			LastCheckAt: now,
		}
	}

	for path, e := range st.Pinned {
		ref, err := swarm.ReferenceFromHex(e.Reference)
		if err != nil {
			continue
		}
		ok, err := client.API.IsRetrievable(context.Background(), ref)
		if err != nil {
			ok = false
		}
		e.LastCheckOK = ok
		e.LastCheckAt = now
		st.Pinned[path] = e
		if !ok {
			fmt.Fprintf(os.Stderr, "[%d] WARN %s → %s not retrievable\n", now, path, e.Reference)
		}
	}

	if err := save(st); err != nil {
		return err
	}
	printStatus(st)
	return nil
}

func listFiles(dir string) ([]string, error) {
	var out []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Mode().IsRegular() {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func printStatus(st *state) {
	total := len(st.Pinned)
	ok := 0
	for _, e := range st.Pinned {
		if e.LastCheckOK {
			ok++
		}
	}
	fmt.Printf("status: %d/%d retrievable\n", ok, total)
}

func load() *state {
	bytes, err := os.ReadFile(stateFile)
	if err != nil {
		return &state{Pinned: map[string]pinnedEntry{}}
	}
	var s state
	if err := json.Unmarshal(bytes, &s); err != nil {
		return &state{Pinned: map[string]pinnedEntry{}}
	}
	if s.Pinned == nil {
		s.Pinned = map[string]pinnedEntry{}
	}
	return &s
}

func save(s *state) error {
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
