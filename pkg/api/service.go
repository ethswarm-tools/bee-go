package api

import (
	"net/http"
	"net/url"
)

// Service is the cross-cutting API endpoint group: pin, tag,
// stewardship, grantee, envelope, and "is reference retrievable?"
// checks. Get one from the top-level Client.API field rather than
// constructing it directly.
type Service struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewService wires up an api.Service against a Bee base URL and HTTP
// client. The top-level bee.NewClient calls this for you; use it
// directly only if you need the api endpoints in isolation (without the
// rest of the bee-go sub-services).
func NewService(baseURL *url.URL, httpClient *http.Client) *Service {
	return &Service{baseURL: baseURL, httpClient: httpClient}
}
