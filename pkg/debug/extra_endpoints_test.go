package debug_test

import (
	"context"
	"encoding/base64"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

func TestService_WelcomeMessage(t *testing.T) {
	var posted string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/welcome-message" && r.Method == http.MethodGet:
			w.Write([]byte(`{"welcomeMessage":"hello swarm"}`))
		case r.URL.Path == "/welcome-message" && r.Method == http.MethodPost:
			b, _ := io.ReadAll(r.Body)
			posted = string(b)
			w.Write([]byte(`{"status":"ok"}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)

	got, err := c.GetWelcomeMessage(context.Background())
	if err != nil {
		t.Fatalf("GetWelcomeMessage: %v", err)
	}
	if got != "hello swarm" {
		t.Errorf("got = %q", got)
	}

	if err := c.SetWelcomeMessage(context.Background(), "new msg"); err != nil {
		t.Fatalf("SetWelcomeMessage: %v", err)
	}
	if !strings.Contains(posted, `"welcomeMessage":"new msg"`) {
		t.Errorf("posted body = %s", posted)
	}
}

func TestService_Loggers(t *testing.T) {
	exp := "one/name"
	encoded := base64.StdEncoding.EncodeToString([]byte(exp))
	var putHit string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := `{"tree":{},"loggers":[{"logger":"a","verbosity":"info","subsystem":"x","id":"1"}]}`
		switch {
		case r.URL.Path == "/loggers" && r.Method == http.MethodGet:
			w.Write([]byte(body))
		case r.URL.Path == "/loggers/"+encoded && r.Method == http.MethodGet:
			w.Write([]byte(body))
		case r.URL.Path == "/loggers/"+encoded && r.Method == http.MethodPut:
			putHit = r.URL.Path
			w.WriteHeader(200)
		default:
			w.WriteHeader(404)
		}
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)

	got, err := c.GetLoggers(context.Background())
	if err != nil {
		t.Fatalf("GetLoggers: %v", err)
	}
	if len(got.Loggers) != 1 || got.Loggers[0].Logger != "a" {
		t.Errorf("loggers = %+v", got.Loggers)
	}

	got2, err := c.GetLoggersByExpression(context.Background(), exp)
	if err != nil {
		t.Fatalf("GetLoggersByExpression: %v", err)
	}
	if len(got2.Loggers) != 1 {
		t.Errorf("filtered loggers = %+v", got2.Loggers)
	}

	if err := c.SetLoggerVerbosity(context.Background(), exp); err != nil {
		t.Fatalf("SetLoggerVerbosity: %v", err)
	}
	if putHit != "/loggers/"+encoded {
		t.Errorf("PUT path = %q want %q", putHit, "/loggers/"+encoded)
	}
}

func TestService_GetLastChequesForPeer(t *testing.T) {
	body := `{
		"peer": "abc",
		"lastreceived": {"beneficiary":"0xb1","chequebook":"0xc1","payout":"100"},
		"lastsent": {"beneficiary":"0xb2","chequebook":"0xc2","payout":"50"}
	}`
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chequebook/cheque/abc" && r.Method == http.MethodGet {
			w.Write([]byte(body))
			return
		}
		w.WriteHeader(404)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)
	got, err := c.GetLastChequesForPeer(context.Background(), "abc")
	if err != nil {
		t.Fatalf("GetLastChequesForPeer: %v", err)
	}
	if got.Peer != "abc" {
		t.Errorf("peer = %q", got.Peer)
	}
	if got.LastReceived == nil || got.LastReceived.Payout.Int64() != 100 {
		t.Errorf("lastReceived = %+v", got.LastReceived)
	}
	if got.LastSent == nil || got.LastSent.Beneficiary != "0xb2" {
		t.Errorf("lastSent = %+v", got.LastSent)
	}
}

func TestService_CashoutOps(t *testing.T) {
	getBody := `{
		"peer": "abc",
		"uncashedAmount": "777",
		"transactionHash": "0xtx",
		"lastCashedCheque": {"beneficiary":"0xb","chequebook":"0xc","payout":"500"},
		"result": {"recipient":"0xr","lastPayout":"500","bounced":false}
	}`
	gotGasPrice := ""
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/chequebook/cashout/abc" && r.Method == http.MethodGet:
			w.Write([]byte(getBody))
		case r.URL.Path == "/chequebook/cashout/abc" && r.Method == http.MethodPost:
			gotGasPrice = r.Header.Get("gas-price")
			w.Write([]byte(`{"transactionHash":"0xcashout"}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)

	la, err := c.GetLastCashoutAction(context.Background(), "abc")
	if err != nil {
		t.Fatalf("GetLastCashoutAction: %v", err)
	}
	if la.UncashedAmount == nil || la.UncashedAmount.Int64() != 777 {
		t.Errorf("uncashedAmount = %v", la.UncashedAmount)
	}
	if la.Result == nil || la.Result.LastPayout == nil || la.Result.LastPayout.Int64() != 500 {
		t.Errorf("result.lastPayout = %+v", la.Result)
	}
	if la.LastCashedCheque == nil || la.LastCashedCheque.Payout.Int64() != 500 {
		t.Errorf("lastCashedCheque = %+v", la.LastCashedCheque)
	}

	hash, err := c.CashoutLastCheque(context.Background(), "abc", big.NewInt(1234))
	if err != nil {
		t.Fatalf("CashoutLastCheque: %v", err)
	}
	if hash != "0xcashout" {
		t.Errorf("hash = %q", hash)
	}
	if gotGasPrice != "1234" {
		t.Errorf("gas-price header = %q", gotGasPrice)
	}
}

func TestService_HealthAndVersions(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Write([]byte(`{"status":"ok","version":"` + debug.SupportedBeeVersionExact + `","apiVersion":"` + debug.SupportedAPIVersion + `"}`))
		case "/":
			w.WriteHeader(200)
		default:
			w.WriteHeader(404)
		}
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)

	h, err := c.GetHealth(context.Background())
	if err != nil || h.Status != "ok" || h.Version != debug.SupportedBeeVersionExact {
		t.Fatalf("GetHealth: %+v err=%v", h, err)
	}

	v, err := c.GetVersions(context.Background())
	if err != nil || v.BeeVersion != debug.SupportedBeeVersionExact || v.SupportedBeeAPIVersion != debug.SupportedAPIVersion {
		t.Errorf("GetVersions: %+v err=%v", v, err)
	}

	exact, err := c.IsSupportedExactVersion(context.Background())
	if err != nil || !exact {
		t.Errorf("IsSupportedExactVersion: %v err=%v", exact, err)
	}

	api, err := c.IsSupportedAPIVersion(context.Background())
	if err != nil || !api {
		t.Errorf("IsSupportedAPIVersion: %v err=%v", api, err)
	}

	if !c.IsConnected(context.Background()) {
		t.Errorf("IsConnected = false")
	}
	if err := c.CheckConnection(context.Background()); err != nil {
		t.Errorf("CheckConnection: %v", err)
	}
}

func TestService_IsSupportedAPIVersion_majorMismatch(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok","version":"x","apiVersion":"99.0.0"}`))
	}))
	defer s.Close()
	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)
	ok, err := c.IsSupportedAPIVersion(context.Background())
	if err != nil || ok {
		t.Errorf("IsSupportedAPIVersion = %v err=%v, want false nil", ok, err)
	}
}

func TestService_IsGateway(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{"gateway-on", 200, `{"gateway":true}`, true},
		{"gateway-off-404", 404, "", false},
		{"gateway-off-flag", 200, `{"gateway":false}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/gateway" {
					w.WriteHeader(tc.status)
					if tc.body != "" {
						w.Write([]byte(tc.body))
					}
					return
				}
				w.WriteHeader(404)
			}))
			defer s.Close()
			u, _ := url.Parse(s.URL)
			c := debug.NewService(u, http.DefaultClient)
			got, err := c.IsGateway(context.Background())
			if err != nil {
				t.Fatalf("IsGateway: %v", err)
			}
			if got != tc.want {
				t.Errorf("got = %v want %v", got, tc.want)
			}
		})
	}
}
