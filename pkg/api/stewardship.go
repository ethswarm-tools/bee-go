package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ethersphere/bee-go/pkg/swarm"
)

// Reupload re-uploads locally pinned data.
func (s *Service) Reupload(ctx context.Context, ref swarm.Reference, batchID string) error {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("stewardship/%s", ref.Value)})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("swarm-postage-batch-id", batchID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("reupload failed with status: %d", resp.StatusCode)
	}
	return nil
}

// IsRetrievable checks if the content is retrievable.
func (s *Service) IsRetrievable(ctx context.Context, ref swarm.Reference) (bool, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("stewardship/%s", ref.Value)})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return false, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("is retrievable check failed with status: %d", resp.StatusCode)
	}

	var res struct {
		IsRetrievable bool `json:"isRetrievable"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return false, err
	}
	return res.IsRetrievable, nil
}
