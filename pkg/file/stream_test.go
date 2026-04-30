package file_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ethersphere/bee-go/pkg/file"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

// HashCollectionEntries should produce a stable, content-addressed
// reference — same input → same output across runs, no HTTP at all.
func TestHashCollectionEntries_Deterministic(t *testing.T) {
	entries := []file.CollectionEntry{
		{Path: "index.html", Data: []byte("<h1>hi</h1>")},
		{Path: "img/logo.png", Data: bytes.Repeat([]byte{0xAB}, 8000)},
	}
	a, err := file.HashCollectionEntries(entries)
	if err != nil {
		t.Fatalf("HashCollectionEntries: %v", err)
	}
	b, err := file.HashCollectionEntries(entries)
	if err != nil {
		t.Fatalf("HashCollectionEntries (2nd): %v", err)
	}
	if a.Hex() != b.Hex() {
		t.Errorf("non-deterministic: %s vs %s", a.Hex(), b.Hex())
	}
	if a.Hex() == strings.Repeat("00", 32) {
		t.Errorf("zero ref")
	}
}

// HashDirectory should match HashCollectionEntries for the same files.
func TestHashDirectory_MatchesEntries(t *testing.T) {
	dir := t.TempDir()
	files := map[string][]byte{
		"a.txt":     []byte("alpha"),
		"sub/b.txt": []byte("beta"),
	}
	for p, d := range files {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, d, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	gotDir, err := file.HashDirectory(dir)
	if err != nil {
		t.Fatalf("HashDirectory: %v", err)
	}
	// Build the same entries explicitly (paths use forward slashes).
	entries := []file.CollectionEntry{
		{Path: "a.txt", Data: []byte("alpha")},
		{Path: "sub/b.txt", Data: []byte("beta")},
	}
	gotEntries, err := file.HashCollectionEntries(entries)
	if err != nil {
		t.Fatalf("HashCollectionEntries: %v", err)
	}
	if gotDir.Hex() != gotEntries.Hex() {
		t.Errorf("HashDirectory ref %s != HashCollectionEntries ref %s", gotDir.Hex(), gotEntries.Hex())
	}
}

// StreamCollectionEntries should upload exactly the chunks the chunker
// emits plus the manifest nodes, end with the manifest root reference,
// and the root must match HashCollectionEntries on the same input.
func TestStreamCollectionEntries(t *testing.T) {
	entries := []file.CollectionEntry{
		{Path: "small.txt", Data: []byte("small")},
		{Path: "big.bin", Data: bytes.Repeat([]byte{0x55}, swarm.ChunkSize+200)}, // 2 leaves
	}
	wantRoot, err := file.HashCollectionEntries(entries)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	var (
		mu        sync.Mutex
		uploaded  int
		uploadFn  = func(_ context.Context, body []byte) (swarm.Reference, error) {
			addr, err := swarm.CalculateChunkAddress(body)
			if err != nil {
				return swarm.Reference{}, err
			}
			return swarm.NewReference(addr)
		}
	)
	_ = uploadFn

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chunks" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		addr, err := swarm.CalculateChunkAddress(body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		uploaded++
		mu.Unlock()
		ref, err := swarm.NewReference(addr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"reference":"` + ref.Hex() + `"}`))
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	svc := file.NewService(u, http.DefaultClient)

	progressCalls := 0
	opts := &file.StreamOptions{
		OnProgress: func(p file.UploadProgress) { progressCalls++ },
	}
	res, err := svc.StreamCollectionEntries(context.Background(), swarm.MustBatchID(strings.Repeat("aa", 32)), entries, opts)
	if err != nil {
		t.Fatalf("StreamCollectionEntries: %v", err)
	}
	if res.Reference.Hex() != wantRoot.Hex() {
		t.Errorf("root = %s want %s", res.Reference.Hex(), wantRoot.Hex())
	}
	// Expect at least: 1 leaf for small.txt + 2 leaves + 1 parent for big.bin
	// + manifest nodes (≥ 1). Don't pin exact count to avoid coupling to
	// mantaray fan-out.
	if uploaded < 5 {
		t.Errorf("too few chunks uploaded: %d", uploaded)
	}
	// Progress must have fired at least once (file-chunk emissions only,
	// not manifest nodes).
	if progressCalls == 0 {
		t.Errorf("OnProgress never called")
	}
}
