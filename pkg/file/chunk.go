package file

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ethersphere/bee-go/pkg/api"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

// UploadChunk uploads a raw chunk to Swarm.
// The data must be at most 4096 bytes + span.
func (s *Service) UploadChunk(ctx context.Context, batchID string, data []byte, opts *api.UploadOptions) (swarm.Reference, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "chunks"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(data))
	if err != nil {
		return swarm.Reference{}, err
	}

	req.Header.Set("Swarm-Postage-Batch-Id", batchID)
	if opts != nil {
		opts.ApplyToRequest(req)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return swarm.Reference{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return swarm.Reference{}, fmt.Errorf("upload chunk failed with status: %d", resp.StatusCode)
	}

	var res struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return swarm.Reference{}, err
	}

	return swarm.Reference{Value: res.Reference}, nil
}

// DownloadChunk downloads a raw chunk from Swarm.
func (s *Service) DownloadChunk(ctx context.Context, ref swarm.Reference) ([]byte, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "chunks/" + ref.Value})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download chunk failed with status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
