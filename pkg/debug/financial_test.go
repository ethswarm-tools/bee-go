package debug_test

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ethswarm-tools/bee-go/pkg/debug"
)

func TestService_Wallet(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/wallet" {
			w.Write([]byte(`{"bzzAddress": "0x1", "nativeAddress": "0x2", "chequebook": "0x3", "bzzBalance": "1000", "nativeTokenBalance": "2000"}`))
			return
		}
		if r.URL.Path == "/wallet/withdraw/bzz" {
			w.Write([]byte(`{"transactionHash": "0xhash3"}`))
			return
		}
		if r.URL.Path == "/wallet/withdraw/nativetoken" {
			w.Write([]byte(`{"transactionHash": "0xhash4"}`))
			return
		}
		if r.URL.Path == "/chequebook/balance" {
			w.Write([]byte(`{"totalBalance": "100", "availableBalance": "50"}`))
			return
		}
		if r.URL.Path == "/chequebook/deposit" {
			w.Write([]byte(`{"transactionHash": "0xhash1"}`))
			return
		}
		if r.URL.Path == "/chequebook/withdraw" {
			w.Write([]byte(`{"transactionHash": "0xhash2"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)

	// Wallet
	// Wallet
	w, err := c.GetWallet(context.Background())
	if err != nil {
		t.Fatalf("GetWallet error = %v", err)
	}
	if w.BzzAddress != "0x1" {
		t.Errorf("BzzAddress = %v, want 0x1", w.BzzAddress)
	}
	// Check new fields if mocked
	if w.BzzBalance != nil && w.BzzBalance.Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("BzzBalance = %v, want 1000", w.BzzBalance)
	}

	// WithdrawBZZ
	txBzz, err := c.WithdrawBZZ(context.Background(), big.NewInt(10), "0xaddr")
	if err != nil {
		t.Fatalf("WithdrawBZZ error = %v", err)
	}
	if txBzz != "0xhash3" {
		t.Errorf("WithdrawBZZ tx = %v, want 0xhash3", txBzz)
	}

	// WithdrawDAI
	txDai, err := c.WithdrawDAI(context.Background(), big.NewInt(10), "0xaddr")
	if err != nil {
		t.Fatalf("WithdrawDAI error = %v", err)
	}
	if txDai != "0xhash4" {
		t.Errorf("WithdrawDAI tx = %v, want 0xhash4", txDai)
	}

	// Balance
	bal, err := c.GetChequebookBalance(context.Background())
	if err != nil {
		t.Fatalf("GetChequebookBalance error = %v", err)
	}
	if bal.TotalBalance.String() != "100" {
		t.Errorf("TotalBalance = %v, want 100", bal.TotalBalance)
	}

	// Deposit
	tx, err := c.DepositTokens(context.Background(), big.NewInt(10))
	if err != nil {
		t.Fatalf("DepositTokens error = %v", err)
	}
	if tx != "0xhash1" {
		t.Errorf("Deposit tx = %v, want 0xhash1", tx)
	}

	// Withdraw
	tx2, err := c.WithdrawTokens(context.Background(), big.NewInt(10))
	if err != nil {
		t.Fatalf("WithdrawTokens error = %v", err)
	}
	if tx2 != "0xhash2" {
		t.Errorf("Withdraw tx = %v, want 0xhash2", tx2)
	}
}

func TestService_Settlements(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/settlements" {
			w.Write([]byte(`{"totalReceived": "100", "totalSent": "50", "settlements": [{"peer": "p1", "received": "10", "sent": "5"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)
	res, err := c.Settlements(context.Background())
	if err != nil {
		t.Fatalf("Settlements error = %v", err)
	}
	if res.TotalReceived.Cmp(big.NewInt(100)) != 0 {
		t.Errorf("TotalReceived = %v, want 100", res.TotalReceived)
	}
	if len(res.Settlements) != 1 || res.Settlements[0].Peer != "p1" {
		t.Errorf("Settlements[0] = %v, want p1", res.Settlements[0])
	}
}

func TestService_LastCheques(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chequebook/cheque" {
			w.Write([]byte(`{"lastcheques": [{"peer": "p1", "lastreceived": {"beneficiary": "b1", "chequebook": "c1", "payout": "50"}}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)
	res, err := c.LastCheques(context.Background())
	if err != nil {
		t.Fatalf("LastCheques error = %v", err)
	}
	if len(res.LastCheques) != 1 || res.LastCheques[0].Peer != "p1" {
		t.Errorf("LastCheques[0] = %v, want p1", res.LastCheques[0])
	}
	if res.LastCheques[0].LastReceived.Payout.Cmp(big.NewInt(50)) != 0 {
		t.Errorf("Payout = %v, want 50", res.LastCheques[0].LastReceived.Payout)
	}
}

func TestService_PendingTransactions(t *testing.T) {
	// Bee actually returns []TransactionInfo objects, not bare hash strings.
	body := `{
		"pendingTransactions": [
			{"transactionHash": "tx1", "to": "0x1", "nonce": 1, "gasPrice": "100", "gasLimit": 21000, "data": "0x", "created": "2026-04-30T00:00:00Z", "description": "stamps", "value": "0"},
			{"transactionHash": "tx2", "to": "0x2", "nonce": 2, "gasPrice": "200", "gasLimit": 21000, "data": "0x", "created": "2026-04-30T00:00:00Z", "description": "stamps", "value": "0"}
		]
	}`
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/transactions" {
			w.Write([]byte(body))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)
	res, err := c.PendingTransactions(context.Background())
	if err != nil {
		t.Fatalf("PendingTransactions error = %v", err)
	}
	if len(res) != 2 || res[0] != "tx1" || res[1] != "tx2" {
		t.Errorf("PendingTransactions = %v, want [tx1, tx2]", res)
	}

	// Also exercise GetAllPendingTransactions / GetPendingTransaction.
	all, err := c.GetAllPendingTransactions(context.Background())
	if err != nil {
		t.Fatalf("GetAllPendingTransactions: %v", err)
	}
	if all[0].GasPrice == nil || all[0].GasPrice.Int64() != 100 {
		t.Errorf("GasPrice = %v", all[0].GasPrice)
	}
}

func TestService_TransactionLifecycle(t *testing.T) {
	rebroadcasted, cancelled := "", ""
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/transactions/0xtx" && r.Method == http.MethodGet:
			w.Write([]byte(`{"transactionHash": "0xtx", "to": "0x1", "nonce": 1, "gasPrice": "100", "gasLimit": 21000, "data": "0x", "created": "", "description": "", "value": "0"}`))
		case r.URL.Path == "/transactions/0xtx" && r.Method == http.MethodPost:
			rebroadcasted = "yes"
			w.Write([]byte(`{"transactionHash": "0xtx-new"}`))
		case r.URL.Path == "/transactions/0xtx" && r.Method == http.MethodDelete:
			cancelled = r.Header.Get("gas-price")
			w.Write([]byte(`{"transactionHash": "0xtx-cancelled"}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)

	info, err := c.GetPendingTransaction(context.Background(), "0xtx")
	if err != nil || info.TransactionHash != "0xtx" {
		t.Errorf("GetPendingTransaction = %+v err=%v", info, err)
	}

	got, err := c.RebroadcastPendingTransaction(context.Background(), "0xtx")
	if err != nil || got != "0xtx-new" {
		t.Errorf("Rebroadcast = %q err=%v", got, err)
	}
	if rebroadcasted != "yes" {
		t.Errorf("rebroadcast endpoint not hit")
	}

	got, err = c.CancelPendingTransaction(context.Background(), "0xtx", big.NewInt(123))
	if err != nil || got != "0xtx-cancelled" {
		t.Errorf("Cancel = %q err=%v", got, err)
	}
	if cancelled != "123" {
		t.Errorf("gas-price header = %q", cancelled)
	}
}
