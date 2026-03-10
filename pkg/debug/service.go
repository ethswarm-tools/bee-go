package debug

import (
	"net/http"
	"net/url"
)

// Service handles debug operations.
type Service struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewService creates a new debug service.
func NewService(baseURL *url.URL, httpClient *http.Client) *Service {
	return &Service{baseURL: baseURL, httpClient: httpClient}
}
