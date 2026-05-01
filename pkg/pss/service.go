package pss

import (
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

// Service is the PSS endpoint group: send (HTTP /pss/send/...) and
// subscribe / receive (websocket /pss/subscribe/...). Get one from the
// top-level Client.PSS field rather than constructing it directly. The
// websocket dialer is shared with [Client] so per-Client tweaks (TLS
// roots, proxy) propagate.
type Service struct {
	baseURL    *url.URL
	httpClient *http.Client
	dialer     *websocket.Dialer
}

// NewService wires up a pss.Service against a Bee base URL, HTTP
// client, and websocket dialer. The top-level bee.NewClient calls this
// for you.
func NewService(baseURL *url.URL, httpClient *http.Client, dialer *websocket.Dialer) *Service {
	return &Service{baseURL: baseURL, httpClient: httpClient, dialer: dialer}
}
