package debug_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ethersphere/bee-go/pkg/debug"
)

func TestService_Health(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    bool
		wantErr bool
	}{
		{
			name: "ok",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			want: true,
		},
		{
			name: "error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.handler)
			defer s.Close()

			u, _ := url.Parse(s.URL)
			c := debug.NewService(u, http.DefaultClient)
			got, err := c.Health(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.Health() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Service.Health() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_NodeInfo(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    *debug.NodeInfo
		wantErr bool
	}{
		{
			name: "ok",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"beeMode": "light", "chequebookEnabled": true, "swapEnabled": false}`))
			},
			want: &debug.NodeInfo{
				BeeMode:           "light",
				ChequebookEnabled: true,
				SwapEnabled:       false,
			},
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "invalid json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`invalid json`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.handler)
			defer s.Close()

			u, _ := url.Parse(s.URL)
			c := debug.NewService(u, http.DefaultClient)
			got, err := c.NodeInfo(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.NodeInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.BeeMode != tt.want.BeeMode {
					t.Errorf("Service.NodeInfo() BeeMode = %v, want %v", got.BeeMode, tt.want.BeeMode)
				}
				if got.ChequebookEnabled != tt.want.ChequebookEnabled {
					t.Errorf("Service.NodeInfo() ChequebookEnabled = %v, want %v", got.ChequebookEnabled, tt.want.ChequebookEnabled)
				}
				if got.SwapEnabled != tt.want.SwapEnabled {
					t.Errorf("Service.NodeInfo() SwapEnabled = %v, want %v", got.SwapEnabled, tt.want.SwapEnabled)
				}
			}
		})
	}
}

func TestService_ChainState(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chainstate" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(`{"chainTip": 100, "block": 99, "currentPrice": 50}`))
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)
	got, err := c.ChainState(context.Background())
	if err != nil {
		t.Fatalf("ChainState error = %v", err)
	}
	if got.ChainTip != 100 || got.Block != 99 || got.CurrentPrice != 50 {
		t.Errorf("ChainState got = %v", got)
	}
}

func TestService_ReserveState(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reservestate" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(`{"radius": 5, "storageRadius": 4, "commitment": 123456}`))
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)
	got, err := c.ReserveState(context.Background())
	if err != nil {
		t.Fatalf("ReserveState error = %v", err)
	}
	if got.Radius != 5 || got.StorageRadius != 4 || got.Commitment != 123456 {
		t.Errorf("ReserveState got = %v", got)
	}
}

func TestService_Topology(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/topology" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(`{"baseAddr": "0x123", "population": 100, "connected": 50, "timestamp": "2023-01-01", "nnLowWatermark": 10, "depth": 5}`))
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)
	got, err := c.Topology(context.Background())
	if err != nil {
		t.Fatalf("Topology error = %v", err)
	}
	if got.BaseAddr != "0x123" || got.Population != 100 || got.Connected != 50 {
		t.Errorf("Topology got = %v", got)
	}
}

func TestService_Peers(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/peers" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(`{"peers": [{"address": "0xabc", "fullNode": true}]}`))
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)
	got, err := c.Peers(context.Background())
	if err != nil {
		t.Fatalf("Peers error = %v", err)
	}
	if len(got.Peers) != 1 || got.Peers[0].Address != "0xabc" || !got.Peers[0].FullNode {
		t.Errorf("Peers got = %v", got)
	}
}

func TestService_Addresses(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/addresses" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(`{"overlay": "0x111", "underlay": ["/ip4/127.0.0.1"], "ethereum": "0xeth", "publicKey": "0xpub", "pssPublicKey": "0xpss"}`))
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)
	got, err := c.Addresses(context.Background())
	if err != nil {
		t.Fatalf("Addresses error = %v", err)
	}
	if got.Overlay != "0x111" || got.Ethereum != "0xeth" || len(got.Underlay) != 1 {
		t.Errorf("Addresses got = %v", got)
	}
}
