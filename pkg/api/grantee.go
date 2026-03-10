package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// GranteesResponse represents the list of grantees.
type GranteesResponse struct {
	Grantees []string `json:"grantees"`
}

// GranteeResponse represents the response from create/patch grantee.
type GranteeResponse struct {
	Ref        string `json:"ref"`
	HistoryRef string `json:"historyref"`
}

// GetGrantees retrieves the grantees for a reference.
func (s *Service) GetGrantees(ctx context.Context, ref string) ([]string, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("grantee/%s", ref)})
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
		return nil, fmt.Errorf("get grantees failed with status: %d", resp.StatusCode)
	}

	var res GranteesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Grantees, nil
}

// CreateGrantees creates a new grantee list.
func (s *Service) CreateGrantees(ctx context.Context, batchID string, grantees []string) (GranteeResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "grantee"})

	body := struct {
		Grantees []string `json:"grantees"`
	}{
		Grantees: grantees,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return GranteeResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(data))
	if err != nil {
		return GranteeResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Swarm-Postage-Batch-Id", batchID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return GranteeResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return GranteeResponse{}, fmt.Errorf("create grantees failed with status: %d", resp.StatusCode)
	}

	var res GranteeResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return GranteeResponse{}, err
	}
	return res, nil
}

// PatchGrantees updates the grantees for a reference.
func (s *Service) PatchGrantees(ctx context.Context, batchID string, ref string, historyRef string, add []string, revoke []string) (GranteeResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("grantee/%s", ref)})

	body := struct {
		Add    []string `json:"add,omitempty"`
		Revoke []string `json:"revoke,omitempty"`
	}{
		Add:    add,
		Revoke: revoke,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return GranteeResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u.String(), bytes.NewReader(data))
	if err != nil {
		return GranteeResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Swarm-Postage-Batch-Id", batchID)
	req.Header.Set("Swarm-Act-History-Address", historyRef)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return GranteeResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return GranteeResponse{}, fmt.Errorf("patch grantees failed with status: %d", resp.StatusCode)
	}

	var res GranteeResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return GranteeResponse{}, err
	}
	return res, nil
}
