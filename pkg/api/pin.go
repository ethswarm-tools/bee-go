package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ethersphere/bee-go/pkg/swarm"
)

// Pin pins a reference.
func (s *Service) Pin(ctx context.Context, ref swarm.Reference) error {
	u := s.baseURL.ResolveReference(&url.URL{Path: "pins/" + ref.Value})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("pin failed with status: %d", resp.StatusCode)
	}
	return nil
}

// Unpin unpins a reference.
func (s *Service) Unpin(ctx context.Context, ref swarm.Reference) error {
	u := s.baseURL.ResolveReference(&url.URL{Path: "pins/" + ref.Value})
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unpin failed with status: %d", resp.StatusCode)
	}
	return nil
}

// GetPin checks the pin status (or gets pin info if it exists).
// For now, simpler check.
// Note: GET /pins/ref generally returns 200/404.
func (s *Service) GetPin(ctx context.Context, ref swarm.Reference) (bool, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "pins/" + ref.Value})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return false, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, fmt.Errorf("get pin status failed with status: %d", resp.StatusCode)
}

// ListPins retrieves all pinned references.
func (s *Service) ListPins(ctx context.Context) ([]swarm.Reference, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "pins"})
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
		return nil, fmt.Errorf("list pins failed with status: %d", resp.StatusCode)
	}

	var res struct {
		References []swarm.Reference `json:"references"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.References, nil
}
