package swarm_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/ethersphere/bee-go/pkg/swarm"
)

// Single-chunk file (≤ ChunkSize bytes): root must equal the CAC.
func TestFileChunker_SingleChunk(t *testing.T) {
	payload := []byte("hello swarm")
	want, err := swarm.MakeContentAddressedChunk(payload)
	if err != nil {
		t.Fatalf("CAC: %v", err)
	}

	var emitted []swarm.Chunk
	c := swarm.NewFileChunker(func(c swarm.Chunk) error {
		emitted = append(emitted, c)
		return nil
	})
	if _, err := c.Write(payload); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := c.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if !bytes.Equal(got.Address.Raw(), want.Address.Raw()) {
		t.Errorf("root = %x want %x", got.Address.Raw(), want.Address.Raw())
	}
	// Exactly one chunk emitted: the leaf == root.
	if len(emitted) != 1 {
		t.Errorf("emitted = %d, want 1", len(emitted))
	}
	if !bytes.Equal(emitted[0].Address.Raw(), want.Address.Raw()) {
		t.Errorf("leaf addr mismatch")
	}
}

// Multi-leaf file (just over one chunk): root must be a level-1 chunk
// over the leaves with total span = file size.
func TestFileChunker_TwoLeaves(t *testing.T) {
	// 4097 bytes → leaf0 (4096) + leaf1 (1) + parent over the two refs.
	payload := bytes.Repeat([]byte{0x42}, swarm.ChunkSize+1)

	var emitted []swarm.Chunk
	c := swarm.NewFileChunker(func(c swarm.Chunk) error {
		emitted = append(emitted, c)
		return nil
	})
	if _, err := c.Write(payload); err != nil {
		t.Fatalf("Write: %v", err)
	}
	root, err := c.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	if len(emitted) != 3 {
		t.Fatalf("emitted = %d, want 3 (2 leaves + 1 root)", len(emitted))
	}
	// First two emissions are the leaves; third is the parent (= root).
	if !bytes.Equal(emitted[2].Address.Raw(), root.Address.Raw()) {
		t.Errorf("last emit should equal root")
	}
	// Root span = total file size.
	if binary.LittleEndian.Uint64(root.Span[:]) != uint64(len(payload)) {
		t.Errorf("root span = %d, want %d", binary.LittleEndian.Uint64(root.Span[:]), len(payload))
	}
	// Root payload is concat of leaf addresses.
	if len(emitted[2].Payload) != 2*swarm.SegmentSize {
		t.Errorf("root payload len = %d, want %d", len(emitted[2].Payload), 2*swarm.SegmentSize)
	}
	if !bytes.Equal(emitted[2].Payload[:32], emitted[0].Address.Raw()) ||
		!bytes.Equal(emitted[2].Payload[32:], emitted[1].Address.Raw()) {
		t.Errorf("root payload doesn't match emitted leaves")
	}
}

// Streaming write in many small pieces should yield the same address as
// a single Write of the same bytes.
func TestFileChunker_StreamingMatchesAtomic(t *testing.T) {
	payload := bytes.Repeat([]byte{0xAB, 0xCD}, 5000) // 10000 bytes, > ChunkSize

	atomic := swarm.NewFileChunker(nil)
	if _, err := atomic.Write(payload); err != nil {
		t.Fatalf("Write: %v", err)
	}
	a, err := atomic.Finalize()
	if err != nil {
		t.Fatalf("Finalize atomic: %v", err)
	}

	streamed := swarm.NewFileChunker(nil)
	for i := 0; i < len(payload); i += 7 {
		end := min(i+7, len(payload))
		if _, err := streamed.Write(payload[i:end]); err != nil {
			t.Fatalf("Write streamed: %v", err)
		}
	}
	b, err := streamed.Finalize()
	if err != nil {
		t.Fatalf("Finalize streamed: %v", err)
	}
	if !bytes.Equal(a.Address.Raw(), b.Address.Raw()) {
		t.Errorf("streaming address differs from atomic: %x vs %x", a.Address.Raw(), b.Address.Raw())
	}
}

// Files larger than 128 leaf chunks need a 3-level tree.
func TestFileChunker_ThreeLevels(t *testing.T) {
	// 129 leaves → level-0 has 128 (collapses) + 1 partial. Level-1 has
	// one full from the collapse + one from the trailing partial that
	// finalize collapses, then level-2 root.
	payload := bytes.Repeat([]byte{0x33}, swarm.ChunkSize*129)

	c := swarm.NewFileChunker(nil)
	if _, err := c.Write(payload); err != nil {
		t.Fatalf("Write: %v", err)
	}
	root, err := c.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if binary.LittleEndian.Uint64(root.Span[:]) != uint64(len(payload)) {
		t.Errorf("root span = %d, want %d", binary.LittleEndian.Uint64(root.Span[:]), len(payload))
	}
}

func TestFileChunker_EmptyFinalize(t *testing.T) {
	c := swarm.NewFileChunker(nil)
	if _, err := c.Finalize(); err == nil {
		t.Errorf("Finalize on empty chunker should error")
	}
}
