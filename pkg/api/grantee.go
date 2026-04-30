package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ethersphere/bee-go/pkg/swarm"
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
func (s *Service) GetGrantees(ctx context.Context, ref swarm.Reference) ([]string, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("grantee/%s", ref.Hex())})
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

	var res GranteesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Grantees, nil
}

// CreateGrantees creates a new grantee list.
func (s *Service) CreateGrantees(ctx context.Context, batchID swarm.BatchID, grantees []string) (GranteeResponse, error) {
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
	req.Header.Set("Swarm-Postage-Batch-Id", batchID.Hex())

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return GranteeResponse{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return GranteeResponse{}, err
	}

	var res GranteeResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return GranteeResponse{}, err
	}
	return res, nil
}

// PatchGrantees updates the grantees for a reference.
func (s *Service) PatchGrantees(ctx context.Context, batchID swarm.BatchID, ref swarm.Reference, historyRef swarm.Reference, add []string, revoke []string) (GranteeResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("grantee/%s", ref.Hex())})

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
	req.Header.Set("Swarm-Postage-Batch-Id", batchID.Hex())
	req.Header.Set("Swarm-Act-History-Address", historyRef.Hex())

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return GranteeResponse{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return GranteeResponse{}, err
	}

	var res GranteeResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return GranteeResponse{}, err
	}
	return res, nil
}
