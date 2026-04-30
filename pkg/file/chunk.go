package file

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

// UploadChunk uploads a single raw chunk (span + payload, up to 4096 bytes
// of payload).
func (s *Service) UploadChunk(ctx context.Context, batchID swarm.BatchID, data []byte, opts *api.UploadOptions) (api.UploadResult, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "chunks"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(data))
	if err != nil {
		return api.UploadResult{}, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	api.PrepareUploadHeaders(req, batchID, opts)

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

// DownloadChunk fetches a single chunk's bytes.
func (s *Service) DownloadChunk(ctx context.Context, ref swarm.Reference, opts *api.DownloadOptions) ([]byte, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "chunks/" + ref.Hex()})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	api.PrepareDownloadHeaders(req, opts)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return nil, err
	}
	return io.ReadAll(resp.Body)
}
