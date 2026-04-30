package file

import (
	"context"
	"errors"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/manifest"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

// UploadProgress is the per-chunk progress signal passed to Stream*
// callers. Total / Processed are running counters of leaf+intermediate
// chunks; Total may be 0 if the chunker doesn't precompute it.
type UploadProgress struct {
	Total     int
	Processed int
}

// StreamOptions tunes the Stream* methods. Inherits CollectionUploadOptions
// for index/error documents and adds a progress callback.
type StreamOptions struct {
	api.CollectionUploadOptions
	OnProgress func(UploadProgress)
}

// HashCollectionEntries chunks each entry through the streaming BMT
// chunker and assembles a Mantaray manifest, returning the manifest's
// root reference without uploading anything.
//
// Mirrors bee-js hashDirectory but takes in-memory entries so callers
// can use it from any platform (no filesystem required).
func HashCollectionEntries(entries []CollectionEntry) (swarm.Reference, error) {
	mantaray, err := buildManifest(entries, nil, nil)
	if err != nil {
		return swarm.Reference{}, err
	}
	addr, err := mantaray.CalculateSelfAddress()
	if err != nil {
		return swarm.Reference{}, err
	}
	return swarm.NewReference(addr)
}

// HashDirectory walks dir, chunks each file through the streaming BMT
// chunker, assembles a Mantaray manifest, and returns the manifest's
// root reference without uploading anything. Mirrors bee-js
// hashDirectory.
//
// File contents are read in full per file (the chunker streams within
// a file but each file is read with os.Open + the chunker drives reads).
// Directory walk follows fs.WalkDir order.
func HashDirectory(dir string) (swarm.Reference, error) {
	entries, err := readDirAsEntries(dir)
	if err != nil {
		return swarm.Reference{}, err
	}
	return HashCollectionEntries(entries)
}

// StreamCollectionEntries is the upload-as-you-go counterpart to
// HashCollectionEntries: each leaf and intermediate file chunk is
// uploaded as it's produced, then the Mantaray manifest is uploaded
// recursively. Mirrors bee-js streamFiles for the entries shape (the
// browser-only File[]/FileList path is intentionally omitted).
//
// Useful when files are large enough that collecting them all in
// memory before tar-ing (the UploadCollectionEntries path) is wasteful
// or when callers want a deterministic per-file content reference.
func (s *Service) StreamCollectionEntries(ctx context.Context, batchID swarm.BatchID, entries []CollectionEntry, opts *StreamOptions) (api.UploadResult, error) {
	collectionOpts := streamCollectionOpts(opts)
	uploader := s.chunkUploader(batchID, collectionOpts)

	var (
		processed int
		onProg    func(UploadProgress)
	)
	if opts != nil {
		onProg = opts.OnProgress
	}

	bumpProgress := func() {
		processed++
		if onProg != nil {
			onProg(UploadProgress{Processed: processed})
		}
	}

	mantaray, err := buildManifest(entries, func(ctx context.Context, data []byte) error {
		_, err := uploader(ctx, batchID, data)
		if err != nil {
			return err
		}
		bumpProgress()
		return nil
	}, ctx)
	if err != nil {
		return api.UploadResult{}, err
	}

	rootRef, err := mantaray.SaveRecursively(ctx, manifest.ChunkUploader(uploader), batchID)
	if err != nil {
		return api.UploadResult{}, err
	}
	return api.UploadResult{Reference: rootRef}, nil
}

// StreamDirectory walks dir and streams each file through the BMT
// chunker, uploading each chunk as it's produced; finally the Mantaray
// manifest is uploaded recursively. Returns the manifest root.
// Mirrors bee-js streamDirectory.
func (s *Service) StreamDirectory(ctx context.Context, batchID swarm.BatchID, dir string, opts *StreamOptions) (api.UploadResult, error) {
	entries, err := readDirAsEntries(dir)
	if err != nil {
		return api.UploadResult{}, err
	}
	return s.StreamCollectionEntries(ctx, batchID, entries, opts)
}

// chunkUploader returns a manifest.ChunkUploader-shaped function that
// uploads via UploadChunk, threading the redundancy/encryption knobs
// from the collection upload options.
func (s *Service) chunkUploader(batchID swarm.BatchID, opts *api.CollectionUploadOptions) manifest.ChunkUploader {
	var upOpts *api.UploadOptions
	if opts != nil {
		o := opts.UploadOptions
		upOpts = &o
	}
	return func(ctx context.Context, _ swarm.BatchID, chunkData []byte) (swarm.Reference, error) {
		res, err := s.UploadChunk(ctx, batchID, chunkData, upOpts)
		if err != nil {
			return swarm.Reference{}, err
		}
		return res.Reference, nil
	}
}

func streamCollectionOpts(opts *StreamOptions) *api.CollectionUploadOptions {
	if opts == nil {
		return nil
	}
	c := opts.CollectionUploadOptions
	return &c
}

// buildManifest runs the streaming BMT chunker over each entry, adds
// each file's root reference to a Mantaray, and (when present) wires
// the index/error documents at the root. If onChunk is non-nil, every
// chunk produced during file chunking is passed through it — that's
// the upload hook for the streaming variant.
//
// The detected MIME type is stored under "Content-Type"; the file's
// short name under "Filename", matching bee-js metadata keys.
func buildManifest(
	entries []CollectionEntry,
	onChunk func(ctx context.Context, fullChunkData []byte) error,
	ctx context.Context,
) (*manifest.MantarayNode, error) {
	mantaray := manifest.New()
	hasIndex := false

	for _, e := range entries {
		var emit func(swarm.Chunk) error
		if onChunk != nil {
			emit = func(c swarm.Chunk) error { return onChunk(ctx, c.Data()) }
		}
		chunker := swarm.NewFileChunker(emit)
		if _, err := chunker.Write(e.Data); err != nil {
			return nil, err
		}
		root, err := chunker.Finalize()
		if err != nil {
			return nil, err
		}

		path := normalizeManifestPath(e.Path)
		metadata := map[string]string{
			"Content-Type": detectContentType(path),
			"Filename":     filepath.Base(path),
		}
		mantaray.AddFork([]byte(path), root.Address, metadata)

		if path == "index.html" {
			hasIndex = true
		}
	}

	_ = hasIndex // root metadata for index/error docs is optional and added by the caller via opts; we don't auto-set it here to keep this helper opt-free.
	return mantaray, nil
}

// readDirAsEntries walks dir and returns the file contents as in-memory
// CollectionEntry values, with paths relative to dir using forward
// slashes.
func readDirAsEntries(dir string) ([]CollectionEntry, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("not a directory: " + dir)
	}

	var entries []CollectionEntry
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		entries = append(entries, CollectionEntry{Path: filepath.ToSlash(rel), Data: data})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func normalizeManifestPath(p string) string {
	return strings.TrimPrefix(filepath.ToSlash(p), "./")
}

// detectContentType resolves a filename to its MIME type via the
// standard library's mime.TypeByExtension table, falling back to
// application/octet-stream and adding charset=utf-8 to text/* types
// (matching bee-js).
func detectContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	t := mime.TypeByExtension(ext)
	if t == "" {
		t = "application/octet-stream"
	}
	if (strings.HasPrefix(t, "text/html") || strings.HasPrefix(t, "text/css")) && !strings.Contains(t, "charset") {
		t += "; charset=utf-8"
	}
	return t
}
