package manifest

import (
	"context"

	"github.com/ethersphere/bee-go/pkg/swarm"
)

// ChunkUploader is anything that can upload a single chunk's full
// (span || payload) bytes and return its reference. We accept the
// interface here rather than depending on pkg/file directly so the
// manifest package stays free of HTTP concerns and tests can stub
// the uploader cleanly.
type ChunkUploader func(ctx context.Context, batchID swarm.BatchID, chunkData []byte) (swarm.Reference, error)

// SaveRecursively walks the trie depth-first, marshaling each node
// into chunks via swarm.FileChunker (a single leaf chunk for nodes
// that fit; an intermediate-level BMT for any that exceed ChunkSize),
// uploading every produced chunk through uploader, and recording the
// root reference in SelfAddress so the parent's marshal can address it.
//
// Returns the root node's reference. Mirrors bee-js
// MantarayNode.saveRecursively.
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

	var uploadErr error
	chunker := swarm.NewFileChunker(func(c swarm.Chunk) error {
		if _, err := uploader(ctx, batchID, c.Data()); err != nil {
			uploadErr = err
			return err
		}
		return nil
	})
	if _, err := chunker.Write(data); err != nil {
		if uploadErr != nil {
			return swarm.Reference{}, uploadErr
		}
		return swarm.Reference{}, err
	}
	root, err := chunker.Finalize()
	if err != nil {
		if uploadErr != nil {
			return swarm.Reference{}, uploadErr
		}
		return swarm.Reference{}, err
	}
	n.SelfAddress = root.Address.Raw()
	return root.Address, nil
}
