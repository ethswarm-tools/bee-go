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

// TransactionInfo describes a pending Bee transaction. Mirrors bee-js
// TransactionInfo. Created is the RFC3339 string Bee emits — left as a
// string so the caller can parse it with whatever timezone handling
// they prefer.
type TransactionInfo struct {
	TransactionHash string   `json:"transactionHash"`
	To              string   `json:"to"`
	Nonce           uint64   `json:"nonce"`
	GasPrice        *big.Int `json:"-"`
	GasLimit        uint64   `json:"gasLimit"`
	GasTipBoost     int      `json:"gasTipBoost"`
	GasTipCap       *big.Int `json:"-"`
	GasFeeCap       *big.Int `json:"-"`
	Data            string   `json:"data"`
	Created         string   `json:"created"`
	Description     string   `json:"description"`
	Value           *big.Int `json:"-"`
}

type transactionInfoJSON struct {
	TransactionHash string `json:"transactionHash"`
	To              string `json:"to"`
	Nonce           uint64 `json:"nonce"`
	GasPrice        string `json:"gasPrice"`
	GasLimit        uint64 `json:"gasLimit"`
	GasTipBoost     int    `json:"gasTipBoost"`
	GasTipCap       string `json:"gasTipCap"`
	GasFeeCap       string `json:"gasFeeCap"`
	Data            string `json:"data"`
	Created         string `json:"created"`
	Description     string `json:"description"`
	Value           string `json:"value"`
}

// UnmarshalJSON handles the big.Int-as-string fields that Bee emits.
func (t *TransactionInfo) UnmarshalJSON(b []byte) error {
	var v transactionInfoJSON
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	t.TransactionHash = v.TransactionHash
	t.To = v.To
	t.Nonce = v.Nonce
	t.GasLimit = v.GasLimit
	t.GasTipBoost = v.GasTipBoost
	t.Data = v.Data
	t.Created = v.Created
	t.Description = v.Description
	t.GasPrice = parseBigInt(v.GasPrice)
	t.GasTipCap = parseBigInt(v.GasTipCap)
	t.GasFeeCap = parseBigInt(v.GasFeeCap)
	t.Value = parseBigInt(v.Value)
	return nil
}

func parseBigInt(s string) *big.Int {
	if s == "" {
		return nil
	}
	v := new(big.Int)
	if _, ok := v.SetString(s, 10); !ok {
		return nil
	}
	return v
}

// GetAllPendingTransactions returns every pending transaction the Bee
// node knows about. Mirrors bee-js Bee.getAllPendingTransactions.
func (s *Service) GetAllPendingTransactions(ctx context.Context) ([]TransactionInfo, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "transactions"})
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
		PendingTransactions []TransactionInfo `json:"pendingTransactions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.PendingTransactions, nil
}

// GetPendingTransaction returns the info for a single pending
// transaction by its hash. Mirrors bee-js Bee.getPendingTransaction.
func (s *Service) GetPendingTransaction(ctx context.Context, txHash string) (TransactionInfo, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("transactions/%s", txHash)})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return TransactionInfo{}, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return TransactionInfo{}, err
	}
	defer resp.Body.Close()
	if err := swarm.CheckResponse(resp); err != nil {
		return TransactionInfo{}, err
	}
	var res TransactionInfo
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return TransactionInfo{}, err
	}
	return res, nil
}

// RebroadcastPendingTransaction replays a pending transaction to the
// network. Useful when the original drops out of mempool. Mirrors
// bee-js Bee.rebroadcastPendingTransaction.
func (s *Service) RebroadcastPendingTransaction(ctx context.Context, txHash string) (string, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("transactions/%s", txHash)})
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
		TransactionHash string `json:"transactionHash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.TransactionHash, nil
}

// CancelPendingTransaction cancels a pending transaction by replacing it
// with a zero-value tx at the same nonce. gasPrice is optional (nil =
// let Bee pick) — when non-nil it's sent in the gas-price header so the
// replacement bumps the fee. Mirrors bee-js Bee.cancelPendingTransaction.
func (s *Service) CancelPendingTransaction(ctx context.Context, txHash string, gasPrice *big.Int) (string, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("transactions/%s", txHash)})
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
	if err != nil {
		return "", err
	}
	if gasPrice != nil {
		req.Header.Set("gas-price", gasPrice.String())
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
		TransactionHash string `json:"transactionHash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.TransactionHash, nil
}
