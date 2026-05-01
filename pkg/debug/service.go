package debug

import (
	"net/http"
	"net/url"
)

// Service is the operator / observability endpoint group: health,
// versions, peers, accounting, chequebook, stake, transactions,
// loggers, and chain / reserve / redistribution state. Get one from
// the top-level Client.Debug field rather than constructing it directly.
type Service struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewService wires up a debug.Service against a Bee base URL and HTTP
// client. The top-level bee.NewClient calls this for you.
func NewService(baseURL *url.URL, httpClient *http.Client) *Service {
	return &Service{baseURL: baseURL, httpClient: httpClient}
}
