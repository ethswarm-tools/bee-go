package swarm

import (
	"net/http"
	"net/url"
)

// Service is the small set of primitive endpoints owned by pkg/swarm
// (most pkg/swarm functionality is offline and exposed as free
// functions / typed-bytes constructors). Get one from the top-level
// Client.Swarm field rather than constructing it directly.
type Service struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewService wires up a swarm.Service against a Bee base URL and HTTP
// client. The top-level bee.NewClient calls this for you.
func NewService(baseURL *url.URL, httpClient *http.Client) *Service {
	return &Service{baseURL: baseURL, httpClient: httpClient}
}
