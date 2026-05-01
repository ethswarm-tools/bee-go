package postage

import (
	"net/http"
	"net/url"
)

// Service is the postage-batch endpoint group: create, top-up, dilute,
// list owned (/stamps) and chain-wide (/batches). Get one from the
// top-level Client.Postage field rather than constructing it directly.
//
// Stamp math is exposed as free functions in this package (GetStampCost,
// GetAmountForDuration, GetDepthForSize, …) since it has no I/O.
type Service struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewService wires up a postage.Service against a Bee base URL and HTTP
// client. The top-level bee.NewClient calls this for you.
func NewService(baseURL *url.URL, httpClient *http.Client) *Service {
	return &Service{baseURL: baseURL, httpClient: httpClient}
}
