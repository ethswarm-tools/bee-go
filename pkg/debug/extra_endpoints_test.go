package debug_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ethersphere/bee-go/pkg/debug"
)

func TestService_GetAccounting(t *testing.T) {
	body := `{
		"peerData": {
			"peer-1": {
				"balance": "100",
				"consumedBalance": "50",
				"thresholdReceived": "1000",
				"thresholdGiven": "1000",
				"currentThresholdReceived": "950",
				"currentThresholdGiven": "950",
				"surplusBalance": "0",
				"reservedBalance": "10",
				"shadowReservedBalance": "5",
				"ghostBalance": "0"
			}
		}
	}`
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/accounting" && r.Method == http.MethodGet {
			w.Write([]byte(body))
			return
		}
		w.WriteHeader(404)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)
	got, err := c.GetAccounting(context.Background())
	if err != nil {
		t.Fatalf("GetAccounting: %v", err)
	}
	p, ok := got["peer-1"]
	if !ok {
		t.Fatalf("peer-1 missing: %v", got)
	}
	if p.Balance == nil || p.Balance.Int64() != 100 {
		t.Errorf("Balance = %v", p.Balance)
	}
	if p.ReservedBalance == nil || p.ReservedBalance.Int64() != 10 {
		t.Errorf("ReservedBalance = %v", p.ReservedBalance)
	}
}

func TestService_StatusPeers(t *testing.T) {
	body := `{
		"snapshots": [
			{"overlay": "abc", "proximity": 12, "beeMode": "full", "reserveSize": 1000, "connectedPeers": 50, "isReachable": true, "lastSyncedBlock": 12345, "committedDepth": 8},
			{"overlay": "def", "proximity": 14, "requestFailed": true}
		]
	}`
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status/peers" && r.Method == http.MethodGet {
			w.Write([]byte(body))
			return
		}
		w.WriteHeader(404)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)
	got, err := c.StatusPeers(context.Background())
	if err != nil {
		t.Fatalf("StatusPeers: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(got))
	}
	if got[0].Overlay != "abc" || got[0].BeeMode != "full" || got[0].ReserveSize != 1000 {
		t.Errorf("snapshot[0] = %+v", got[0])
	}
	if !got[1].RequestFailed {
		t.Errorf("snapshot[1] should have RequestFailed=true: %+v", got[1])
	}
}

func TestService_StatusNeighborhoods(t *testing.T) {
	body := `{"neighborhoods": [
		{"neighborhood": "1010", "reserveSizeWithinRadius": 100, "proximity": 4},
		{"neighborhood": "1100", "reserveSizeWithinRadius": 200, "proximity": 4}
	]}`
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status/neighborhoods" && r.Method == http.MethodGet {
			w.Write([]byte(body))
			return
		}
		w.WriteHeader(404)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)
	got, err := c.StatusNeighborhoods(context.Background())
	if err != nil {
		t.Fatalf("StatusNeighborhoods: %v", err)
	}
	if len(got) != 2 || got[0].Neighborhood != "1010" || got[0].ReserveSizeWithinRadius != 100 {
		t.Errorf("got = %+v", got)
	}
}

func TestService_ConnectPeer(t *testing.T) {
	gotPath := ""
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method == http.MethodPost {
			w.Write([]byte(`{"address": "0xoverlay"}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)

	// Bee escapes the multiaddress as one path segment; gorilla/mux on the
	// server side accepts the leading-slash form. Our client strips leading
	// slashes so callers can pass either shape.
	addr := "/dns/bee.example.com/tcp/1634/p2p/16Uiu2HAm"
	overlay, err := c.ConnectPeer(context.Background(), addr)
	if err != nil {
		t.Fatalf("ConnectPeer: %v", err)
	}
	if overlay != "0xoverlay" {
		t.Errorf("overlay = %q", overlay)
	}
	// No double slashes after /connect/.
	if gotPath == "" || gotPath[:9] != "/connect/" {
		t.Errorf("path = %q", gotPath)
	}
}
