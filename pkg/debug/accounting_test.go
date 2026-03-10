package debug_test

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethersphere/bee-go/pkg/debug"
)

func TestService_Accounting(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/balances" {
			w.Write([]byte(`{"balances": [{"peer": "p1", "balance": "100"}]}`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/balances/") {
			w.Write([]byte(`{"peer": "p1", "balance": "100"}`))
			return
		}
		if r.URL.Path == "/consumed" {
			w.Write([]byte(`{"balances": [{"peer": "p1", "balance": "50"}]}`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/consumed/") {
			w.Write([]byte(`{"peer": "p1", "balance": "50"}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)

	// GetBalances
	balances, err := c.GetBalances(context.Background())
	if err != nil {
		t.Fatalf("GetBalances error = %v", err)
	}
	if len(balances) != 1 || balances[0].Balance.Cmp(big.NewInt(100)) != 0 {
		t.Errorf("GetBalances = %v, want 100", balances)
	}

	// GetPeerBalance
	balance, err := c.GetPeerBalance(context.Background(), "p1")
	if err != nil {
		t.Fatalf("GetPeerBalance error = %v", err)
	}
	if balance.Balance.Cmp(big.NewInt(100)) != 0 {
		t.Errorf("GetPeerBalance = %v, want 100", balance)
	}

	// GetConsumed
	consumed, err := c.GetConsumed(context.Background())
	if err != nil {
		t.Fatalf("GetConsumed error = %v", err)
	}
	if len(consumed) != 1 || consumed[0].Balance.Cmp(big.NewInt(50)) != 0 {
		t.Errorf("GetConsumed = %v, want 50", consumed)
	}

	// GetPeerConsumed
	peerConsumed, err := c.GetPeerConsumed(context.Background(), "p1")
	if err != nil {
		t.Fatalf("GetPeerConsumed error = %v", err)
	}
	if peerConsumed.Balance.Cmp(big.NewInt(50)) != 0 {
		t.Errorf("GetPeerConsumed = %v, want 50", peerConsumed)
	}
}
