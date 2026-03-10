package swarm

import (
	"net/http"
	"net/url"
)

// Service handles swarm primitive operations.
type Service struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewService creates a new swarm service.
func NewService(baseURL *url.URL, httpClient *http.Client) *Service {
	return &Service{baseURL: baseURL, httpClient: httpClient}
}
