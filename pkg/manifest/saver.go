package manifest

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ethersphere/bee-go/pkg/swarm"
)

// ChunkUploader is anything that can upload a single chunk's full
// (span || payload) bytes and return its reference. We accept the
// interface here rather than depending on pkg/file directly so the
// manifest package stays free of HTTP concerns and tests can stub
// the uploader cleanly.
type ChunkUploader func(ctx context.Context, batchID swarm.BatchID, chunkData []byte) (swarm.Reference, error)

// SaveRecursively walks the trie depth-first, marshaling each node
// into a chunk body, calling uploader to publish that chunk, and
// recording the resulting reference in the node's SelfAddress so the
// parent's marshal can reference it.
//
// Returns the root node's reference. Mirrors bee-js
// MantarayNode.saveRecursively.
//
// LIMITATION: each marshaled node must fit in a single chunk
// (ChunkSize bytes). Mantaray's trie structure keeps individual
// nodes small in practice — a node carries at most 256 forks — but
// extreme cases would need multi-chunk node serialization, tracked
// alongside the same limitation in CalculateSelfAddress.
func (n *MantarayNode) SaveRecursively(ctx context.Context, uploader ChunkUploader, batchID swarm.BatchID) (swarm.Reference, error) {
	for _, fork := range n.Forks {
		if len(fork.Node.SelfAddress) > 0 {
			continue
		}
		ref, err := fork.Node.SaveRecursively(ctx, uploader, batchID)
		if err != nil {
			return swarm.Reference{}, err
		}
		fork.Node.SelfAddress = ref.Raw()
	}

	data, err := n.Marshal()
	if err != nil {
		return swarm.Reference{}, err
	}
	if len(data) > swarm.ChunkSize {
		return swarm.Reference{}, fmt.Errorf("manifest: marshaled node size %d exceeds single chunk; multi-chunk BMT not yet implemented", len(data))
	}

	span := make([]byte, swarm.SpanSize)
	binary.LittleEndian.PutUint64(span, uint64(len(data)))
	body := make([]byte, 0, swarm.SpanSize+len(data))
	body = append(body, span...)
	body = append(body, data...)

	ref, err := uploader(ctx, batchID, body)
	if err != nil {
		return swarm.Reference{}, err
	}
	n.SelfAddress = ref.Raw()
	return ref, nil
}
