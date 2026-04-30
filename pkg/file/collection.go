package file

import (
	"archive/tar"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/ethersphere/bee-go/pkg/api"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

// CollectionEntry is one file in an in-memory collection. Path is the
// relative tar entry path (e.g. "index.html" or "assets/logo.png"); Data
// is the file contents.
//
// Mirrors the bee-js Collection entry shape (path + data) without the
// browser-only File object — Go callers supply bytes directly.
type CollectionEntry struct {
	Path string
	Data []byte
}

// UploadCollectionEntries packages the given entries as a tar stream and
// uploads them via POST /bzz, same as UploadCollection but without
// touching the filesystem. Useful for programmatic site generation, test
// fixtures, or callers that already hold the files in memory.
//
// Mirrors bee-js makeCollectionFromFileList + bzz.uploadCollection.
func (s *Service) UploadCollectionEntries(ctx context.Context, batchID swarm.BatchID, entries []CollectionEntry, opts *api.CollectionUploadOptions) (api.UploadResult, error) {
	pr, pw := io.Pipe()
	go func() {
		tw := tar.NewWriter(pw)
		for _, e := range entries {
			hdr := &tar.Header{
				Name: e.Path,
				Mode: 0o644,
				Size: int64(len(e.Data)),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				pw.CloseWithError(err)
				return
			}
			if _, err := tw.Write(e.Data); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
		if err := tw.Close(); err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.Close()
	}()

	u := s.baseURL.ResolveReference(&url.URL{Path: "bzz"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), pr)
	if err != nil {
		return api.UploadResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-tar")
	req.Header.Set("Swarm-Collection", "true")
	api.PrepareCollectionUploadHeaders(req, batchID, opts)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return api.UploadResult{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return api.UploadResult{}, err
	}

	var res struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return api.UploadResult{}, err
	}
	return api.ReadUploadResult(res.Reference, resp.Header)
}

// CollectionSize returns the cumulative byte size of the entries.
// Mirrors bee-js getCollectionSize.
func CollectionSize(entries []CollectionEntry) int64 {
	var total int64
	for _, e := range entries {
		total += int64(len(e.Data))
	}
	return total
}
