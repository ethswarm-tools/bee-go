package debug_test

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethswarm-tools/bee-go/pkg/debug"
)

func TestService_Staking(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stake" {
			if r.Method == http.MethodGet {
				w.Write([]byte(`{"stakedAmount": "1000"}`))
				return
			}
			if r.Method == http.MethodDelete {
				w.Write([]byte(`{"txHash": "0xmigrate"}`))
				return
			}
		}
		if strings.HasPrefix(r.URL.Path, "/stake/") {
			if r.Method == http.MethodPost {
				w.Write([]byte(`{"txHash": "0xstake"}`))
				return
			}
		}
		if r.URL.Path == "/stake/withdrawable" {
			if r.Method == http.MethodGet {
				w.Write([]byte(`{"withdrawableAmount": "500"}`))
				return
			}
			if r.Method == http.MethodDelete {
				w.Write([]byte(`{"txHash": "0xwithdraw"}`))
				return
			}
		}
		if r.URL.Path == "/redistributionstate" {
			w.Write([]byte(`{"minimumGasFunds": "100", "hasSufficientFunds": true, "round": 5}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)

	// GetStake
	stake, err := c.GetStake(context.Background())
	if err != nil {
		t.Fatalf("GetStake error = %v", err)
	}
	if stake.Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("GetStake = %v, want 1000", stake)
	}

	// Stake
	tx, err := c.Stake(context.Background(), big.NewInt(100))
	if err != nil {
		t.Fatalf("Stake error = %v", err)
	}
	if tx != "0xstake" {
		t.Errorf("Stake tx = %v, want 0xstake", tx)
	}

	// GetWithdrawableStake
	widthrawable, err := c.GetWithdrawableStake(context.Background())
	if err != nil {
		t.Fatalf("GetWithdrawableStake error = %v", err)
	}
	if widthrawable.Cmp(big.NewInt(500)) != 0 {
		t.Errorf("GetWithdrawableStake = %v, want 500", widthrawable)
	}

	// WithdrawSurplusStake
	tx2, err := c.WithdrawSurplusStake(context.Background())
	if err != nil {
		t.Fatalf("WithdrawSurplusStake error = %v", err)
	}
	if tx2 != "0xwithdraw" {
		t.Errorf("WithdrawSurplusStake tx = %v, want 0xwithdraw", tx2)
	}

	// MigrateStake
	tx3, err := c.MigrateStake(context.Background())
	if err != nil {
		t.Fatalf("MigrateStake error = %v", err)
	}
	if tx3 != "0xmigrate" {
		t.Errorf("MigrateStake tx = %v, want 0xmigrate", tx3)
	}

	// RedistributionState
	state, err := c.RedistributionState(context.Background())
	if err != nil {
		t.Fatalf("RedistributionState error = %v", err)
	}
	if state.Round != 5 || !state.HasSufficientFunds {
		t.Errorf("RedistributionState = %v", state)
	}
}
