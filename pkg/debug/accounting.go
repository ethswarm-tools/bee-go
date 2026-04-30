package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"

	"github.com/ethersphere/bee-go/pkg/swarm"
)

// Balance represents the balance with a peer.
type Balance struct {
	Peer    string   `json:"peer"`
	Balance *big.Int `json:"balance"`
}

type balanceJSON struct {
	Peer    string `json:"peer"`
	Balance string `json:"balance"`
}

func (b *Balance) UnmarshalJSON(data []byte) error {
	var v balanceJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	b.Peer = v.Peer
	b.Balance = new(big.Int)
	b.Balance.SetString(v.Balance, 10)
	return nil
}

// BalancesResponse represents the list of balances.
type BalancesResponse struct {
	Balances []Balance `json:"balances"`
}

// GetBalances retrieves the balances with all known peers.
func (s *Service) GetBalances(ctx context.Context) ([]Balance, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "balances"})
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

	var res BalancesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Balances, nil
}

// GetPeerBalance retrieves the balance with a specific peer.
func (s *Service) GetPeerBalance(ctx context.Context, address string) (*Balance, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("balances/%s", address)})
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

	var res Balance
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetPastDueConsumptionBalances is the bee-js alias for GetConsumed.
// Both hit /consumed and return one entry per peer.
func (s *Service) GetPastDueConsumptionBalances(ctx context.Context) ([]Balance, error) {
	return s.GetConsumed(ctx)
}

// GetPastDueConsumptionPeerBalance is the bee-js alias for GetPeerConsumed.
func (s *Service) GetPastDueConsumptionPeerBalance(ctx context.Context, address string) (*Balance, error) {
	return s.GetPeerConsumed(ctx, address)
}

// GetConsumed retrieves the past due consumption balances with all known peers.
func (s *Service) GetConsumed(ctx context.Context) ([]Balance, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "consumed"})
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

	var res BalancesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Balances, nil
}

// GetPeerConsumed retrieves the past due consumption balance with a specific peer.
func (s *Service) GetPeerConsumed(ctx context.Context, address string) (*Balance, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("consumed/%s", address)})
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

	var res Balance
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}
