package postage

import (
	"net/http"
	"net/url"
)

// Service handles postage operations.
type Service struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewService creates a new postage service.
func NewService(baseURL *url.URL, httpClient *http.Client) *Service {
	return &Service{baseURL: baseURL, httpClient: httpClient}
}
