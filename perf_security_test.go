package bee

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

// TestNewClient_DefaultHTTPTimeout asserts the default *http.Client
// constructed by NewClient bounds requests at DefaultHTTPTimeout. The
// stock net/http client has *no* timeout, which can leave a stuck
// connection hanging forever.
func TestNewClient_DefaultHTTPTimeout(t *testing.T) {
	c, err := NewClient("http://localhost:1633")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if got := c.httpClient.Timeout; got != DefaultHTTPTimeout {
		t.Errorf("default Timeout = %v, want %v", got, DefaultHTTPTimeout)
	}
}

// TestWithHTTPClient_BypassesDefault asserts that supplying a custom
// *http.Client lets the caller take full responsibility for timeouts —
// no surprise injection by the option chain.
func TestWithHTTPClient_BypassesDefault(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	c, err := NewClient("http://localhost:1633", WithHTTPClient(custom))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.httpClient != custom {
		t.Errorf("WithHTTPClient: client not the one we passed")
	}
}

// TestWithToken_AddsAuthorizationHeader verifies the bearer-token
// transport rewrites every outbound request with Authorization: Bearer
// <token> without disturbing other headers.
func TestWithToken_AddsAuthorizationHeader(t *testing.T) {
	var seenAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","version":"x","apiVersion":"8.0.0"}`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, WithToken("s3cr3t"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.Debug.GetHealth(context.Background()); err != nil {
		t.Fatalf("GetHealth: %v", err)
	}
	if seenAuth != "Bearer s3cr3t" {
		t.Errorf("Authorization = %q, want %q", seenAuth, "Bearer s3cr3t")
	}
}

// TestPing_ReturnsRoundTrip verifies Ping issues GET /health and
// returns a sane non-zero duration.
func TestPing_ReturnsRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("unexpected path %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ok", "version": "x", "apiVersion": "8.0.0",
		})
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL)
	d, err := c.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if d <= 0 {
		t.Errorf("Ping duration = %v, want >0", d)
	}
}

// TestPrivateKey_String_Redacts ensures fmt.Stringer doesn't leak the
// scalar through default %v / %s formatting (a common cause of secrets
// ending up in panics or logs).
func TestPrivateKey_String_Redacts(t *testing.T) {
	priv, _ := swarm.PrivateKeyFromHex(strings.Repeat("11", 32))
	s := priv.String()
	if strings.Contains(s, "11") || strings.Contains(s, "1111") {
		t.Errorf("PrivateKey.String leaks key material: %q", s)
	}
	if !strings.Contains(strings.ToLower(s), "redacted") {
		t.Errorf("PrivateKey.String %q does not advertise redaction", s)
	}
}

// TestPrivateKey_Equal_ConstantTimeShape asserts the new Equal method
// returns the same boolean as a byte-equal check. The constant-time
// property itself can't be unit-tested cheaply; this just guards the
// shape so a future refactor doesn't drop the call.
func TestPrivateKey_Equal_Shape(t *testing.T) {
	a, _ := swarm.PrivateKeyFromHex(strings.Repeat("aa", 32))
	b, _ := swarm.PrivateKeyFromHex(strings.Repeat("aa", 32))
	c, _ := swarm.PrivateKeyFromHex(strings.Repeat("bb", 32))
	if !a.Equal(b) {
		t.Errorf("Equal(a, a-clone) = false, want true")
	}
	if a.Equal(c) {
		t.Errorf("Equal(a, b) = true, want false")
	}
}

// TestPrivateKey_Zeroize_WipesBytes asserts Zeroize clears the
// underlying buffer, leaving the value unable to derive the original
// public key.
func TestPrivateKey_Zeroize_WipesBytes(t *testing.T) {
	priv, _ := swarm.PrivateKeyFromHex(strings.Repeat("11", 32))
	priv.Zeroize()
	raw := priv.Raw()
	for i, b := range raw {
		if b != 0 {
			t.Errorf("byte[%d] = %x after Zeroize", i, b)
		}
	}
}

// TestWithToken_StripsOnCrossHostRedirect ensures the bearer-token
// transport DOES NOT resend Authorization to a redirect target on a
// different host. A misbehaving / compromised Bee that responds 302
// to attacker.com would otherwise leak the token to the attacker.
func TestWithToken_StripsOnCrossHostRedirect(t *testing.T) {
	var attackerSawAuth string
	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attackerSawAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","version":"x","apiVersion":"8.0.0"}`))
	}))
	defer attacker.Close()

	bee := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, attacker.URL+"/health", http.StatusFound)
	}))
	defer bee.Close()

	c, err := NewClient(bee.URL, WithToken("s3cr3t"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.Debug.GetHealth(context.Background()); err != nil {
		t.Fatalf("GetHealth: %v", err)
	}
	if attackerSawAuth != "" {
		t.Errorf("token leaked to redirect target: %q", attackerSawAuth)
	}
}

// TestRedactURL_StripsQueryAndFragment verifies the helper used by the
// HTTP logger and error formatter drops query strings and fragments.
func TestRedactURL_StripsQueryAndFragment(t *testing.T) {
	cases := []struct{ in, want string }{
		{"http://bee/path?token=secret", "http://bee/path"},
		{"http://bee/path?a=1&b=2#frag", "http://bee/path"},
		{"http://bee/path", "http://bee/path"},
		{"http://bee/", "http://bee/"},
	}
	for _, tc := range cases {
		u, _ := url.Parse(tc.in)
		if got := swarm.RedactURL(u); got != tc.want {
			t.Errorf("RedactURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestValidateCollectionUploadOptions_RejectsCRLF ensures
// IndexDocument/ErrorDocument with header-injection payloads error
// out instead of silently smuggling onto the wire.
func TestValidateCollectionUploadOptions_RejectsCRLF(t *testing.T) {
	bad := []string{"foo\r\nX-Injected: bar", "foo\nbar", "foo\rbar", "foo\x00bar"}
	for _, v := range bad {
		opts := &api.CollectionUploadOptions{IndexDocument: v}
		if err := api.ValidateCollectionUploadOptions(opts); err == nil {
			t.Errorf("ValidateCollectionUploadOptions accepted %q", v)
		}
		opts = &api.CollectionUploadOptions{ErrorDocument: v}
		if err := api.ValidateCollectionUploadOptions(opts); err == nil {
			t.Errorf("ValidateCollectionUploadOptions accepted %q in ErrorDocument", v)
		}
	}
	// Sanity: nil options and empty strings are valid.
	if err := api.ValidateCollectionUploadOptions(nil); err != nil {
		t.Errorf("nil opts must be valid: %v", err)
	}
	if err := api.ValidateCollectionUploadOptions(&api.CollectionUploadOptions{IndexDocument: "index.html"}); err != nil {
		t.Errorf("plain index.html must be valid: %v", err)
	}
}
