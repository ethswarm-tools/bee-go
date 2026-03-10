package file

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ethersphere/bee-go/pkg/api"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

// UploadData uploads raw data to Swarm.
// Returns the reference of the uploaded data.
func (s *Service) UploadData(ctx context.Context, batchID string, data io.Reader, opts *api.UploadOptions) (swarm.Reference, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "bytes"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), data)
	if err != nil {
		return swarm.Reference{}, err
	}

	req.Header.Set("Swarm-Postage-Batch-Id", batchID)
	// Default to octet-stream if not provided, user can wrap reader to provide content type if needed?
	// For now simple API.
	req.Header.Set("Content-Type", "application/octet-stream")

	if opts != nil {
		opts.ApplyToRequest(req)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return swarm.Reference{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return swarm.Reference{}, fmt.Errorf("upload data failed with status: %d", resp.StatusCode)
	}

	var res struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return swarm.Reference{}, err
	}

	return swarm.Reference{Value: res.Reference}, nil
}

// DownloadData downloads raw data from Swarm.
// Returns a ReadCloser that must be closed by the caller.
func (s *Service) DownloadData(ctx context.Context, ref swarm.Reference) (io.ReadCloser, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "bytes/" + ref.Value})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download data failed with status: %d", resp.StatusCode)
	}

	return resp.Body, nil
}
