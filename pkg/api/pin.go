package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

// PinIntegrity is one row of the GET /pins/check NDJSON stream — the
// integrity status of a single pinned reference. IsHealthy returns
// true iff every chunk under the reference is present and valid.
type PinIntegrity struct {
	Reference swarm.Reference `json:"reference"`
	Total     int             `json:"total"`
	Missing   int             `json:"missing"`
	Invalid   int             `json:"invalid"`
}

// IsHealthy reports whether the pinned reference's chunk tree is
// fully retrievable. False means at least one chunk is missing or
// invalid; the missing / invalid counts pinpoint the breakage.
func (p PinIntegrity) IsHealthy() bool {
	return p.Missing == 0 && p.Invalid == 0
}

// Pin pins a reference.
func (s *Service) Pin(ctx context.Context, ref swarm.Reference) error {
	u := s.baseURL.ResolveReference(&url.URL{Path: "pins/" + ref.Hex()})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return err
	}
	return nil
}

// Unpin unpins a reference.
func (s *Service) Unpin(ctx context.Context, ref swarm.Reference) error {
	u := s.baseURL.ResolveReference(&url.URL{Path: "pins/" + ref.Hex()})
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return err
	}
	return nil
}

// GetPin checks the pin status (or gets pin info if it exists).
// For now, simpler check.
// Note: GET /pins/ref generally returns 200/404.
func (s *Service) GetPin(ctx context.Context, ref swarm.Reference) (bool, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "pins/" + ref.Hex()})
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
	return false, swarm.NewBeeResponseError(http.MethodGet, u.String(), resp)
}

// CheckPins streams the GET /pins/check NDJSON response and collects
// the integrity rows into a slice. ref is optional — when nil, every
// pinned reference is checked; otherwise only the named reference is
// reported (as the ?ref={hex} query parameter). Returns an empty
// slice on success when no rows match.
//
// Mirrors bee-rs ApiService::check_pins and bee-py
// client.api.check_pins.
func (s *Service) CheckPins(ctx context.Context, ref *swarm.Reference) ([]PinIntegrity, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "pins/check"})
	if ref != nil {
		q := u.Query()
		q.Set("ref", ref.Hex())
		u.RawQuery = q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := swarm.CheckResponse(resp); err != nil {
		return nil, err
	}

	out := make([]PinIntegrity, 0)
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var row PinIntegrity
		if err := json.Unmarshal(line, &row); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
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

	if err := swarm.CheckResponse(resp); err != nil {
		return nil, err
	}

	var res struct {
		References []swarm.Reference `json:"references"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.References, nil
}
