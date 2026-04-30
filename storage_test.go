package bee_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	bee "github.com/ethersphere/bee-go"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

// fakeBee mounts the minimum endpoints BuyStorage / GetStorageCost need:
// /chainstate (currentPrice) and /stamps (create batch).
func fakeBee(t *testing.T, currentPrice uint64) *httptest.Server {
	t.Helper()
	const batchHex = "abababababababababababababababababababababababababababababababab"
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/chainstate" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"chainTip":     1000,
				"block":        1000,
				"totalAmount":  "0",
				"currentPrice": fmt.Sprintf("%d", currentPrice),
			})
		case strings.HasPrefix(r.URL.Path, "/stamps/") && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"batchID": batchHex})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestBuyStorage_FlowsThroughChainstate(t *testing.T) {
	s := fakeBee(t, 24000)
	defer s.Close()

	c, err := bee.NewClient(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	size, _ := swarm.SizeFromMegabytes(1)
	dur := swarm.DurationFromDays(7)

	got, err := c.BuyStorage(context.Background(), size, dur, nil)
	if err != nil {
		t.Fatalf("BuyStorage: %v", err)
	}
	if got.Hex() == "" {
		t.Errorf("expected batch ID")
	}
}

func TestGetStorageCost_DependsOnPrice(t *testing.T) {
	s := fakeBee(t, 24000)
	defer s.Close()
	c, _ := bee.NewClient(s.URL)
	size, _ := swarm.SizeFromGigabytes(1)
	dur := swarm.DurationFromDays(30)

	cost, err := c.GetStorageCost(context.Background(), size, dur, nil)
	if err != nil {
		t.Fatalf("GetStorageCost: %v", err)
	}
	if cost.ToPLURBigInt().Sign() <= 0 {
		t.Errorf("cost should be positive, got %s", cost.ToPLURString())
	}
}
