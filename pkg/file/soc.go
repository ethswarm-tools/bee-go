package file

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ethersphere/bee-go/pkg/api"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

// UploadSOC uploads a Single Owner Chunk.
func (s *Service) UploadSOC(ctx context.Context, batchID string, owner string, id string, signature string, data []byte, opts *api.UploadOptions) (swarm.Reference, error) {
	path := fmt.Sprintf("soc/%s/%s", owner, id)
	u := s.baseURL.ResolveReference(&url.URL{Path: path})
	q := u.Query()
	q.Set("sig", signature)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(data))
	if err != nil {
		return swarm.Reference{}, err
	}

	req.Header.Set("Swarm-Postage-Batch-Id", batchID)
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
		return swarm.Reference{}, fmt.Errorf("upload soc failed with status: %d", resp.StatusCode)
	}

	var res struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return swarm.Reference{}, err
	}

	return swarm.Reference{Value: res.Reference}, nil
}
