package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"

	"github.com/ethswarm-tools/bee-go/pkg/swarm"
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

// PeerAccounting describes the full accounting state with one peer:
// settlement balances, the configured + dynamic credit/debit thresholds,
// and the various reserved/surplus/ghost positions Bee tracks. All
// monetary fields are PLUR (BZZ base units).
//
// Bee node endpoint: GET /accounting. Not exposed by bee-js; useful for
// monitoring swap state at finer granularity than /balances.
type PeerAccounting struct {
	Balance                  *big.Int
	ConsumedBalance          *big.Int
	ThresholdReceived        *big.Int
	ThresholdGiven           *big.Int
	CurrentThresholdReceived *big.Int
	CurrentThresholdGiven    *big.Int
	SurplusBalance           *big.Int
	ReservedBalance          *big.Int
	ShadowReservedBalance    *big.Int
	GhostBalance             *big.Int
}

type peerAccountingJSON struct {
	Balance                  string `json:"balance"`
	ConsumedBalance          string `json:"consumedBalance"`
	ThresholdReceived        string `json:"thresholdReceived"`
	ThresholdGiven           string `json:"thresholdGiven"`
	CurrentThresholdReceived string `json:"currentThresholdReceived"`
	CurrentThresholdGiven    string `json:"currentThresholdGiven"`
	SurplusBalance           string `json:"surplusBalance"`
	ReservedBalance          string `json:"reservedBalance"`
	ShadowReservedBalance    string `json:"shadowReservedBalance"`
	GhostBalance             string `json:"ghostBalance"`
}

// UnmarshalJSON handles the bigint-as-string fields Bee emits.
func (p *PeerAccounting) UnmarshalJSON(b []byte) error {
	var v peerAccountingJSON
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	p.Balance = parseAccountingBigInt(v.Balance)
	p.ConsumedBalance = parseAccountingBigInt(v.ConsumedBalance)
	p.ThresholdReceived = parseAccountingBigInt(v.ThresholdReceived)
	p.ThresholdGiven = parseAccountingBigInt(v.ThresholdGiven)
	p.CurrentThresholdReceived = parseAccountingBigInt(v.CurrentThresholdReceived)
	p.CurrentThresholdGiven = parseAccountingBigInt(v.CurrentThresholdGiven)
	p.SurplusBalance = parseAccountingBigInt(v.SurplusBalance)
	p.ReservedBalance = parseAccountingBigInt(v.ReservedBalance)
	p.ShadowReservedBalance = parseAccountingBigInt(v.ShadowReservedBalance)
	p.GhostBalance = parseAccountingBigInt(v.GhostBalance)
	return nil
}

func parseAccountingBigInt(s string) *big.Int {
	if s == "" {
		return nil
	}
	v := new(big.Int)
	if _, ok := v.SetString(s, 10); !ok {
		return nil
	}
	return v
}

// GetAccounting returns per-peer accounting info keyed by peer overlay
// address. Strictly richer than GetBalances. Bee-only endpoint (not in
// bee-js).
func (s *Service) GetAccounting(ctx context.Context) (map[string]PeerAccounting, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "accounting"})
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
		PeerData map[string]PeerAccounting `json:"peerData"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.PeerData, nil
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
