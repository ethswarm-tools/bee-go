package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ethersphere/bee-go/pkg/swarm"
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

	if err := swarm.CheckResponse(resp); err != nil {
		return 0, err
	}

	var res struct {
		DurationSeconds float64 `json:"durationSeconds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, err
	}
	return res.DurationSeconds, nil
}
