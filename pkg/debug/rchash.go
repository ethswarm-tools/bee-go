package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// RCHash retrieves the RCHash estimate.
func (s *Service) RCHash(ctx context.Context, depth int, anchor1 string, anchor2 string) (float64, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("rchash/%d/%s/%s", depth, anchor1, anchor2)})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("rchash failed with status: %d", resp.StatusCode)
	}

	var res struct {
		DurationSeconds float64 `json:"durationSeconds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, err
	}
	return res.DurationSeconds, nil
}
