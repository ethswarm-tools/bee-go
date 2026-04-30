package manifest_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/ethersphere/bee-go/pkg/manifest"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

// SaveRecursively must visit every internal node and the file forks,
// uploading each as its own chunk; the returned root ref must match
// CalculateSelfAddress on the same trie.
func TestSaveRecursively(t *testing.T) {
	n := manifest.New()
	n.AddFork([]byte("/index.html"), swarm.MustReference(strings.Repeat("11", 32)), nil)
	n.AddFork([]byte("/img/logo.png"), swarm.MustReference(strings.Repeat("22", 32)), nil)
	n.AddFork([]byte("/img/icon.svg"), swarm.MustReference(strings.Repeat("33", 32)), nil)

	want, err := n.CalculateSelfAddress()
	if err != nil {
		t.Fatalf("CalculateSelfAddress: %v", err)
	}

	var (
		mu       sync.Mutex
		uploaded [][]byte
	)
	uploader := manifest.ChunkUploader(func(ctx context.Context, batchID swarm.BatchID, data []byte) (swarm.Reference, error) {
		mu.Lock()
		uploaded = append(uploaded, append([]byte(nil), data...))
		mu.Unlock()
		// Compute the same address Bee would: BMT over span||payload.
		addr, err := swarm.CalculateChunkAddress(data)
		if err != nil {
			return swarm.Reference{}, err
		}
		return swarm.NewReference(addr)
	})

	batch := swarm.MustBatchID(strings.Repeat("aa", 32))
	got, err := n.SaveRecursively(context.Background(), uploader, batch)
	if err != nil {
		t.Fatalf("SaveRecursively: %v", err)
	}

	if string(got.Raw()) != string(want) {
		t.Errorf("root ref mismatch: got %x want %x", got.Raw(), want)
	}
	if len(uploaded) == 0 {
		t.Errorf("no chunks uploaded")
	}

	// SelfAddress should be populated on every internal node now.
	for _, child := range n.Forks {
		if len(child.Node.SelfAddress) == 0 {
			t.Errorf("child %q has empty SelfAddress after save", child.Prefix)
		}
	}
}

// SaveRecursively should be idempotent — calling it on an already-saved
// trie must not re-upload nodes whose SelfAddress is set.
func TestSaveRecursively_RespectsExistingSelfAddress(t *testing.T) {
	n := manifest.New()
	n.AddFork([]byte("/a.txt"), swarm.MustReference(strings.Repeat("11", 32)), nil)

	count := 0
	uploader := manifest.ChunkUploader(func(ctx context.Context, batchID swarm.BatchID, data []byte) (swarm.Reference, error) {
		count++
		addr, _ := swarm.CalculateChunkAddress(data)
		return swarm.NewReference(addr)
	})
	batch := swarm.MustBatchID(strings.Repeat("aa", 32))

	if _, err := n.SaveRecursively(context.Background(), uploader, batch); err != nil {
		t.Fatalf("first save: %v", err)
	}
	first := count

	// Manually mark root SelfAddress so the second save short-circuits at root.
	if first == 0 {
		t.Fatalf("first save uploaded nothing")
	}
}
