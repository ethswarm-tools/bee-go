package debug

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/url"

	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

// WalletResponse represents the node's wallet addresses and balances.
type WalletResponse struct {
	BzzAddress         string   `json:"bzzAddress"`
	NativeAddress      string   `json:"nativeAddress"`
	Chequebook         string   `json:"chequebook"`
	BzzBalance         *big.Int `json:"bzzBalance"`         // Added
	NativeTokenBalance *big.Int `json:"nativeTokenBalance"` // Added
	ChainID            int64    `json:"chainID"`            // Added
	WalletAddress      string   `json:"walletAddress"`      // Added
}

type walletJSON struct {
	BzzAddress         string `json:"bzzAddress"`
	NativeAddress      string `json:"nativeAddress"`
	Chequebook         string `json:"chequebook"`
	BzzBalance         string `json:"bzzBalance"`
	NativeTokenBalance string `json:"nativeTokenBalance"`
	ChainID            int64  `json:"chainID"`
	WalletAddress      string `json:"walletAddress"`
}

func (w *WalletResponse) UnmarshalJSON(b []byte) error {
	var v walletJSON
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	w.BzzAddress = v.BzzAddress
	w.NativeAddress = v.NativeAddress
	w.Chequebook = v.Chequebook
	w.ChainID = v.ChainID
	w.WalletAddress = v.WalletAddress

	if v.BzzBalance != "" {
		w.BzzBalance = new(big.Int)
		w.BzzBalance.SetString(v.BzzBalance, 10)
	}
	if v.NativeTokenBalance != "" {
		w.NativeTokenBalance = new(big.Int)
		w.NativeTokenBalance.SetString(v.NativeTokenBalance, 10)
	}
	return nil
}

// ChequebookBalanceResponse represents the chequebook balance.
type ChequebookBalanceResponse struct {
	TotalBalance     *big.Int `json:"totalBalance"`
	AvailableBalance *big.Int `json:"availableBalance"`
}

type chequebookBalanceJSON struct {
	TotalBalance     string `json:"totalBalance"`
	AvailableBalance string `json:"availableBalance"`
}

func (c *ChequebookBalanceResponse) UnmarshalJSON(b []byte) error {
	var v chequebookBalanceJSON
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	t := new(big.Int)
	t.SetString(v.TotalBalance, 10)
	c.TotalBalance = t

	a := new(big.Int)
	a.SetString(v.AvailableBalance, 10)
	c.AvailableBalance = a
	return nil
}

// GetWallet retrieves the node's wallet addresses and balances.
func (s *Service) GetWallet(ctx context.Context) (WalletResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "wallet"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return WalletResponse{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return WalletResponse{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return WalletResponse{}, err
	}

	var res WalletResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return WalletResponse{}, err
	}
	return res, nil
}

// WithdrawBZZ withdraws BZZ tokens from the wallet.
func (s *Service) WithdrawBZZ(ctx context.Context, amount *big.Int, address string) (string, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "wallet/withdraw/bzz"})
	q := u.Query()
	q.Set("amount", amount.String())
	q.Set("address", address)
	u.RawQuery = q.Encode()

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

// WithdrawDAI withdraws Native tokens (DAI/ETH) from the wallet.
func (s *Service) WithdrawDAI(ctx context.Context, amount *big.Int, address string) (string, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "wallet/withdraw/nativetoken"})
	q := u.Query()
	q.Set("amount", amount.String())
	q.Set("address", address)
	u.RawQuery = q.Encode()

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

// WithdrawBZZToExternalWallet is the bee-js name for WithdrawBZZ.
func (s *Service) WithdrawBZZToExternalWallet(ctx context.Context, amount *big.Int, address string) (string, error) {
	return s.WithdrawBZZ(ctx, amount, address)
}

// WithdrawDAIToExternalWallet is the bee-js name for WithdrawDAI.
func (s *Service) WithdrawDAIToExternalWallet(ctx context.Context, amount *big.Int, address string) (string, error) {
	return s.WithdrawDAI(ctx, amount, address)
}

// GetChequebookBalance retrieves the chequebook balance.
func (s *Service) GetChequebookBalance(ctx context.Context) (ChequebookBalanceResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "chequebook/balance"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return ChequebookBalanceResponse{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return ChequebookBalanceResponse{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return ChequebookBalanceResponse{}, err
	}

	var res ChequebookBalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return ChequebookBalanceResponse{}, err
	}
	return res, nil
}

// DepositTokens deposits tokens into the chequebook.
func (s *Service) DepositTokens(ctx context.Context, amount *big.Int) (string, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "chequebook/deposit"})
	q := u.Query()
	q.Set("amount", amount.String())
	u.RawQuery = q.Encode()

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

// WithdrawTokens withdraws tokens from the chequebook.
func (s *Service) WithdrawTokens(ctx context.Context, amount *big.Int) (string, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "chequebook/withdraw"})
	q := u.Query()
	q.Set("amount", amount.String())
	u.RawQuery = q.Encode()

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

// DepositBZZToChequebook is the bee-js name for DepositTokens.
func (s *Service) DepositBZZToChequebook(ctx context.Context, amount *big.Int) (string, error) {
	return s.DepositTokens(ctx, amount)
}

// WithdrawBZZFromChequebook is the bee-js name for WithdrawTokens.
func (s *Service) WithdrawBZZFromChequebook(ctx context.Context, amount *big.Int) (string, error) {
	return s.WithdrawTokens(ctx, amount)
}

// Settlement represents a settlement with a peer.
type Settlement struct {
	Peer     string   `json:"peer"`
	Received *big.Int `json:"received"`
	Sent     *big.Int `json:"sent"`
}

type settlementJSON struct {
	Peer     string `json:"peer"`
	Received string `json:"received"`
	Sent     string `json:"sent"`
}

func (s *Settlement) UnmarshalJSON(b []byte) error {
	var v settlementJSON
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	s.Peer = v.Peer
	if v.Received != "" {
		s.Received = new(big.Int)
		s.Received.SetString(v.Received, 10)
	}
	if v.Sent != "" {
		s.Sent = new(big.Int)
		s.Sent.SetString(v.Sent, 10)
	}
	return nil
}

// SettlementsResponse represents list of settlements.
type SettlementsResponse struct {
	TotalReceived *big.Int     `json:"totalReceived"`
	TotalSent     *big.Int     `json:"totalSent"`
	Settlements   []Settlement `json:"settlements"`
}

type settlementsResponseJSON struct {
	TotalReceived string       `json:"totalReceived"`
	TotalSent     string       `json:"totalSent"`
	Settlements   []Settlement `json:"settlements"`
}

func (s *SettlementsResponse) UnmarshalJSON(b []byte) error {
	var v settlementsResponseJSON
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	s.Settlements = v.Settlements
	if v.TotalReceived != "" {
		s.TotalReceived = new(big.Int)
		s.TotalReceived.SetString(v.TotalReceived, 10)
	}
	if v.TotalSent != "" {
		s.TotalSent = new(big.Int)
		s.TotalSent.SetString(v.TotalSent, 10)
	}
	return nil
}

// PeerSettlement retrieves the sent and received settlement totals for
// a specific peer. Mirrors bee-js Bee.getSettlements (note the
// per-peer naming).
func (s *Service) PeerSettlement(ctx context.Context, peer string) (Settlement, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "settlements/" + peer})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Settlement{}, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return Settlement{}, err
	}
	defer resp.Body.Close()
	if err := swarm.CheckResponse(resp); err != nil {
		return Settlement{}, err
	}
	var res Settlement
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return Settlement{}, err
	}
	return res, nil
}

// Settlements retrieves a list of settlements.
func (s *Service) Settlements(ctx context.Context) (SettlementsResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "settlements"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return SettlementsResponse{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return SettlementsResponse{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return SettlementsResponse{}, err
	}

	var res SettlementsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return SettlementsResponse{}, err
	}
	return res, nil
}

type LastCheque struct {
	Peer         string `json:"peer"`
	LastReceived *struct {
		Beneficiary string   `json:"beneficiary"`
		Chequebook  string   `json:"chequebook"`
		Payout      *big.Int `json:"payout"`
	} `json:"lastreceived"`
}

type lastChequeJSON struct {
	Peer         string `json:"peer"`
	LastReceived *struct {
		Beneficiary string `json:"beneficiary"`
		Chequebook  string `json:"chequebook"`
		Payout      string `json:"payout"`
	} `json:"lastreceived"`
}

func (l *LastCheque) UnmarshalJSON(b []byte) error {
	var v lastChequeJSON
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	l.Peer = v.Peer
	if v.LastReceived != nil {
		l.LastReceived = &struct {
			Beneficiary string   `json:"beneficiary"`
			Chequebook  string   `json:"chequebook"`
			Payout      *big.Int `json:"payout"`
		}{
			Beneficiary: v.LastReceived.Beneficiary,
			Chequebook:  v.LastReceived.Chequebook,
		}
		if v.LastReceived.Payout != "" {
			l.LastReceived.Payout = new(big.Int)
			l.LastReceived.Payout.SetString(v.LastReceived.Payout, 10)
		}
	}
	return nil
}

type ChequesResponse struct {
	LastCheques []LastCheque `json:"lastcheques"`
}

// LastCheques retrieves the last cheques for all peers.
func (s *Service) LastCheques(ctx context.Context) (ChequesResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "chequebook/cheque"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return ChequesResponse{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return ChequesResponse{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return ChequesResponse{}, err
	}

	var res ChequesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return ChequesResponse{}, err
	}
	return res, nil
}

// PendingTransactions retrieves the list of pending transaction hashes.
// For full transaction info use GetAllPendingTransactions.
func (s *Service) PendingTransactions(ctx context.Context) ([]string, error) {
	txs, err := s.GetAllPendingTransactions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(txs))
	for _, t := range txs {
		out = append(out, t.TransactionHash)
	}
	return out, nil
}

// Cheque is one peer's last sent or received cheque.
type Cheque struct {
	Beneficiary string   `json:"beneficiary"`
	Chequebook  string   `json:"chequebook"`
	Payout      *big.Int `json:"-"`
}

type chequeJSON struct {
	Beneficiary string `json:"beneficiary"`
	Chequebook  string `json:"chequebook"`
	Payout      string `json:"payout"`
}

// UnmarshalJSON handles the bigint-as-string Payout field.
func (c *Cheque) UnmarshalJSON(b []byte) error {
	var v chequeJSON
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	c.Beneficiary = v.Beneficiary
	c.Chequebook = v.Chequebook
	c.Payout = parseAccountingBigInt(v.Payout)
	return nil
}

// PeerCheques is the response from GET /chequebook/cheque/{peer}: the
// last sent and received cheques for that peer (either may be nil).
type PeerCheques struct {
	Peer         string  `json:"peer"`
	LastReceived *Cheque `json:"lastreceived"`
	LastSent     *Cheque `json:"lastsent"`
}

// GetLastChequesForPeer returns the last sent + received cheques for a
// specific peer overlay. Mirrors bee-js Bee.getLastChequesForPeer.
func (s *Service) GetLastChequesForPeer(ctx context.Context, peer string) (PeerCheques, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "chequebook/cheque/" + peer})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return PeerCheques{}, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return PeerCheques{}, err
	}
	defer resp.Body.Close()
	if err := swarm.CheckResponse(resp); err != nil {
		return PeerCheques{}, err
	}
	var res PeerCheques
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return PeerCheques{}, err
	}
	return res, nil
}

// CashoutResult describes the on-chain outcome of a previous cashout.
type CashoutResult struct {
	Recipient  string   `json:"recipient"`
	LastPayout *big.Int `json:"-"`
	Bounced    bool     `json:"bounced"`
}

type cashoutResultJSON struct {
	Recipient  string `json:"recipient"`
	LastPayout string `json:"lastPayout"`
	Bounced    bool   `json:"bounced"`
}

// UnmarshalJSON handles the bigint-as-string LastPayout field.
func (c *CashoutResult) UnmarshalJSON(b []byte) error {
	var v cashoutResultJSON
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	c.Recipient = v.Recipient
	c.Bounced = v.Bounced
	c.LastPayout = parseAccountingBigInt(v.LastPayout)
	return nil
}

// LastCashoutAction is the response from GET /chequebook/cashout/{peer}.
// LastCashedCheque, TransactionHash and Result are nil when no cashout
// has happened yet for that peer.
type LastCashoutAction struct {
	Peer             string         `json:"peer"`
	UncashedAmount   *big.Int       `json:"-"`
	TransactionHash  *string        `json:"transactionHash"`
	LastCashedCheque *Cheque        `json:"lastCashedCheque"`
	Result           *CashoutResult `json:"result"`
}

type lastCashoutActionJSON struct {
	Peer             string         `json:"peer"`
	UncashedAmount   string         `json:"uncashedAmount"`
	TransactionHash  *string        `json:"transactionHash"`
	LastCashedCheque *Cheque        `json:"lastCashedCheque"`
	Result           *CashoutResult `json:"result"`
}

// UnmarshalJSON handles the bigint-as-string UncashedAmount field.
func (l *LastCashoutAction) UnmarshalJSON(b []byte) error {
	var v lastCashoutActionJSON
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	l.Peer = v.Peer
	l.TransactionHash = v.TransactionHash
	l.LastCashedCheque = v.LastCashedCheque
	l.Result = v.Result
	l.UncashedAmount = parseAccountingBigInt(v.UncashedAmount)
	return nil
}

// GetLastCashoutAction returns the last cashout state (transaction hash,
// cashed cheque, on-chain result) for a peer. Mirrors bee-js
// Bee.getLastCashoutAction.
func (s *Service) GetLastCashoutAction(ctx context.Context, peer string) (LastCashoutAction, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "chequebook/cashout/" + peer})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return LastCashoutAction{}, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return LastCashoutAction{}, err
	}
	defer resp.Body.Close()
	if err := swarm.CheckResponse(resp); err != nil {
		return LastCashoutAction{}, err
	}
	var res LastCashoutAction
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return LastCashoutAction{}, err
	}
	return res, nil
}

// CashoutLastCheque cashes out the last received cheque for the peer.
// gasPrice is optional (nil = let Bee pick); when non-nil it's sent in
// the gas-price header. Returns the cashout transaction hash.
//
// Mirrors bee-js Bee.cashoutLastCheque.
func (s *Service) CashoutLastCheque(ctx context.Context, peer string, gasPrice *big.Int) (string, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "chequebook/cashout/" + peer})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
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
