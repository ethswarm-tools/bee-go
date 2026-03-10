package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// EnvelopeResponse represents the envelope response.
type EnvelopeResponse struct {
	Issuer    string `json:"issuer"`
	Index     string `json:"index"`
	Timestamp string `json:"timestamp"`
	Signature string `json:"signature"`
}

// PostEnvelope posts an envelope.
func (s *Service) PostEnvelope(ctx context.Context, batchID string, ref string) (EnvelopeResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("envelope/%s", ref)})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return EnvelopeResponse{}, err
	}
	req.Header.Set("Swarm-Postage-Batch-Id", batchID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return EnvelopeResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return EnvelopeResponse{}, fmt.Errorf("post envelope failed with status: %d", resp.StatusCode)
	}

	var res EnvelopeResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return EnvelopeResponse{}, err
	}
	return res, nil
}
