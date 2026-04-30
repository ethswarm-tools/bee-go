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

// GetStake retrieves the amount of staked BZZ.
func (s *Service) GetStake(ctx context.Context) (*big.Int, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "stake"})
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
		StakedAmount string `json:"stakedAmount"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	val := new(big.Int)
	if _, ok := val.SetString(res.StakedAmount, 10); !ok {
		return nil, fmt.Errorf("invalid big.Int string: %s", res.StakedAmount)
	}
	return val, nil
}

// Stake stakes a given amount of tokens.
func (s *Service) Stake(ctx context.Context, amount *big.Int) (string, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("stake/%s", amount.String())})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return "", err
	}

	var res struct {
		TxHash string `json:"txHash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.TxHash, nil
}

// GetWithdrawableStake retrieves the amount of withdrawable staked BZZ.
func (s *Service) GetWithdrawableStake(ctx context.Context) (*big.Int, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "stake/withdrawable"})
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
		WithdrawableAmount string `json:"withdrawableAmount"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	val := new(big.Int)
	if _, ok := val.SetString(res.WithdrawableAmount, 10); !ok {
		return nil, fmt.Errorf("invalid big.Int string: %s", res.WithdrawableAmount)
	}
	return val, nil
}

// WithdrawSurplusStake withdraws surplus stake.
func (s *Service) WithdrawSurplusStake(ctx context.Context) (string, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "stake/withdrawable"})
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return "", err
	}

	var res struct {
		TxHash string `json:"txHash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.TxHash, nil
}

// MigrateStake migrates the stake.
func (s *Service) MigrateStake(ctx context.Context) (string, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "stake"})
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return "", err
	}

	var res struct {
		TxHash string `json:"txHash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.TxHash, nil
}

// RedistributionStateResponse represents the redistribution state.
type RedistributionStateResponse struct {
	MinimumGasFunds           *big.Int `json:"minimumGasFunds"`
	HasSufficientFunds        bool     `json:"hasSufficientFunds"`
	IsFrozen                  bool     `json:"isFrozen"`
	IsFullySynced             bool     `json:"isFullySynced"`
	Phase                     string   `json:"phase"`
	Round                     uint64   `json:"round"`
	LastWonRound              uint64   `json:"lastWonRound"`
	LastPlayedRound           uint64   `json:"lastPlayedRound"`
	LastFrozenRound           uint64   `json:"lastFrozenRound"`
	LastSelectedRound         uint64   `json:"lastSelectedRound"`
	LastSampleDurationSeconds uint64   `json:"lastSampleDurationSeconds"`
	Block                     uint64   `json:"block"`
	Reward                    *big.Int `json:"reward"`
	Fees                      *big.Int `json:"fees"`
	IsHealthy                 bool     `json:"isHealthy"`
}

type redistributionStateJSON struct {
	MinimumGasFunds           string `json:"minimumGasFunds"`
	HasSufficientFunds        bool   `json:"hasSufficientFunds"`
	IsFrozen                  bool   `json:"isFrozen"`
	IsFullySynced             bool   `json:"isFullySynced"`
	Phase                     string `json:"phase"`
	Round                     uint64 `json:"round"`
	LastWonRound              uint64 `json:"lastWonRound"`
	LastPlayedRound           uint64 `json:"lastPlayedRound"`
	LastFrozenRound           uint64 `json:"lastFrozenRound"`
	LastSelectedRound         uint64 `json:"lastSelectedRound"`
	LastSampleDurationSeconds uint64 `json:"lastSampleDurationSeconds"`
	Block                     uint64 `json:"block"`
	Reward                    string `json:"reward"`
	Fees                      string `json:"fees"`
	IsHealthy                 bool   `json:"isHealthy"`
}

func (r *RedistributionStateResponse) UnmarshalJSON(b []byte) error {
	var v redistributionStateJSON
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	r.HasSufficientFunds = v.HasSufficientFunds
	r.IsFrozen = v.IsFrozen
	r.IsFullySynced = v.IsFullySynced
	r.Phase = v.Phase
	r.Round = v.Round
	r.LastWonRound = v.LastWonRound
	r.LastPlayedRound = v.LastPlayedRound
	r.LastFrozenRound = v.LastFrozenRound
	r.LastSelectedRound = v.LastSelectedRound
	r.LastSampleDurationSeconds = v.LastSampleDurationSeconds
	r.Block = v.Block
	r.IsHealthy = v.IsHealthy

	if v.MinimumGasFunds != "" {
		r.MinimumGasFunds = new(big.Int)
		r.MinimumGasFunds.SetString(v.MinimumGasFunds, 10)
	}
	if v.Reward != "" {
		r.Reward = new(big.Int)
		r.Reward.SetString(v.Reward, 10)
	}
	if v.Fees != "" {
		r.Fees = new(big.Int)
		r.Fees.SetString(v.Fees, 10)
	}
	return nil
}

// RedistributionState retrieves the redistribution state.
func (s *Service) RedistributionState(ctx context.Context) (RedistributionStateResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "redistributionstate"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return RedistributionStateResponse{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return RedistributionStateResponse{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return RedistributionStateResponse{}, err
	}

	var res RedistributionStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return RedistributionStateResponse{}, err
	}
	return res, nil
}
