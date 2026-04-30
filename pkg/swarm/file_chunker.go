package swarm

import (
	"encoding/binary"
)

// MaxBranches is the fan-out at every intermediate level of the file
// Merkle tree: 4096 / 32 = 128 child references per parent chunk.
const MaxBranches = ChunkSize / SegmentSize

// FileChunker streams bytes into Swarm content-addressed chunks. As
// each leaf or intermediate chunk completes its address is finalised
// and (optionally) emitted via the OnChunk callback. After all input
// is written, Finalize collapses the level stack into a single root
// chunk and returns it.
//
// Mirrors the bee-js MerkleTree streaming chunker (cafe-utility).
//
//	c := swarm.NewFileChunker(func(c swarm.Chunk) error {
//	    return upload(c.Data())
//	})
//	c.Write(filePayload)
//	root, _ := c.Finalize()
type FileChunker struct {
	OnChunk func(Chunk) error

	leafBuf []byte
	// levels[0] holds level-0 (leaf) refs queued for level-1 collapse.
	// levels[i] holds level-i refs queued for level-(i+1) collapse.
	levels [][]levelRef
}

type levelRef struct {
	addr []byte // 32-byte chunk address
	span uint64 // sum of bytes covered by the chunk's subtree
}

// NewFileChunker returns a chunker that calls onChunk for every chunk
// it produces (leaves, intermediates, root). Pass nil to compute the
// root address without emitting anything (offline hash mode).
func NewFileChunker(onChunk func(Chunk) error) *FileChunker {
	return &FileChunker{OnChunk: onChunk}
}

// Write appends bytes to the input stream. Whenever the leaf buffer
// reaches ChunkSize a leaf chunk is sealed and its ref propagated up.
func (fc *FileChunker) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		room := ChunkSize - len(fc.leafBuf)
		take := min(len(p), room)
		fc.leafBuf = append(fc.leafBuf, p[:take]...)
		p = p[take:]
		written += take
		if len(fc.leafBuf) == ChunkSize {
			if err := fc.flushLeaf(); err != nil {
				return written, err
			}
		}
	}
	return written, nil
}

// Finalize seals any trailing partial leaf, climbs the level stack,
// and returns the root chunk. Calling Finalize on an empty chunker
// returns an error — empty payload is not a valid Swarm chunk.
func (fc *FileChunker) Finalize() (Chunk, error) {
	// Trailing partial leaf, or a single full leaf that never collapsed.
	if len(fc.leafBuf) > 0 || (len(fc.levels) == 0 && len(fc.leafBuf) == 0) {
		if len(fc.leafBuf) == 0 {
			return Chunk{}, NewBeeArgumentError("empty payload", 0)
		}
		if err := fc.flushLeaf(); err != nil {
			return Chunk{}, err
		}
	}

	// Collapse the level stack from the bottom up. At each level there
	// may be a partial group (<MaxBranches) — those are sealed too.
	// When a level has exactly one ref *and* every level above it is
	// empty, that ref is the root.
	for level := 0; level < len(fc.levels); level++ {
		// If this is the highest level and it holds exactly one ref, we
		// reached the root — emit nothing more (the leaf or intermediate
		// chunk for that ref was already emitted when it was sealed).
		if level == len(fc.levels)-1 && len(fc.levels[level]) == 1 {
			break
		}
		if len(fc.levels[level]) == 0 {
			continue
		}
		// Seal the (possibly partial) group at this level into a parent.
		if err := fc.collapseLevel(level); err != nil {
			return Chunk{}, err
		}
	}

	rootLevel := len(fc.levels) - 1
	root := fc.levels[rootLevel][0]
	addr, err := NewReference(root.addr)
	if err != nil {
		return Chunk{}, err
	}
	var span [SpanSize]byte
	binary.LittleEndian.PutUint64(span[:], root.span)

	// For the root, we already emitted the chunk during Write or the
	// internal collapses; rebuild the Chunk struct from the level stack.
	// We don't keep the payload around for intermediates, so for files
	// >ChunkSize the returned Chunk has zero-length Payload — callers
	// should rely on Address. The single-leaf case still has Payload.
	c := Chunk{Address: addr, Span: span}
	if rootLevel == 0 && len(fc.leafBuf) > 0 {
		// Only true if Finalize was the first call to seal a single leaf.
		// (Already cleared by flushLeaf above; this branch is defensive.)
		c.Payload = append([]byte(nil), fc.leafBuf...)
	}
	return c, nil
}

// flushLeaf seals the current leaf buffer into a Chunk, emits it, and
// pushes its ref onto level 0.
func (fc *FileChunker) flushLeaf() error {
	if len(fc.leafBuf) == 0 {
		return nil
	}
	payload := fc.leafBuf
	fc.leafBuf = nil

	var span [SpanSize]byte
	binary.LittleEndian.PutUint64(span[:], uint64(len(payload)))

	full := make([]byte, 0, SpanSize+len(payload))
	full = append(full, span[:]...)
	full = append(full, payload...)
	addr, err := CalculateChunkAddress(full)
	if err != nil {
		return err
	}
	ref, err := NewReference(addr)
	if err != nil {
		return err
	}
	if fc.OnChunk != nil {
		c := Chunk{Address: ref, Span: span, Payload: payload}
		if err := fc.OnChunk(c); err != nil {
			return err
		}
	}
	if len(fc.levels) == 0 {
		fc.levels = append(fc.levels, nil)
	}
	fc.levels[0] = append(fc.levels[0], levelRef{addr: addr, span: uint64(len(payload))})
	if len(fc.levels[0]) == MaxBranches {
		return fc.collapseLevel(0)
	}
	return nil
}

// collapseLevel groups every levelRef at level i into a single parent
// chunk (payload = concatenation of their addresses, span = sum of
// their spans), emits the parent, and pushes its ref onto level i+1.
// If level i+1 then reaches MaxBranches it is collapsed too.
func (fc *FileChunker) collapseLevel(level int) error {
	refs := fc.levels[level]
	if len(refs) == 0 {
		return nil
	}
	fc.levels[level] = nil

	payload := make([]byte, 0, len(refs)*SegmentSize)
	var totalSpan uint64
	for _, r := range refs {
		payload = append(payload, r.addr...)
		totalSpan += r.span
	}

	var span [SpanSize]byte
	binary.LittleEndian.PutUint64(span[:], totalSpan)

	full := make([]byte, 0, SpanSize+len(payload))
	full = append(full, span[:]...)
	full = append(full, payload...)
	addr, err := CalculateChunkAddress(full)
	if err != nil {
		return err
	}
	ref, err := NewReference(addr)
	if err != nil {
		return err
	}
	if fc.OnChunk != nil {
		c := Chunk{Address: ref, Span: span, Payload: payload}
		if err := fc.OnChunk(c); err != nil {
			return err
		}
	}
	if level+1 >= len(fc.levels) {
		fc.levels = append(fc.levels, nil)
	}
	fc.levels[level+1] = append(fc.levels[level+1], levelRef{addr: addr, span: totalSpan})
	if len(fc.levels[level+1]) == MaxBranches {
		return fc.collapseLevel(level + 1)
	}
	return nil
}
