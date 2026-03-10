package api

import (
	"net/http"
	"net/url"
)

// Service handles general API operations.
type Service struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewService creates a new API service.
func NewService(baseURL *url.URL, httpClient *http.Client) *Service {
	return &Service{baseURL: baseURL, httpClient: httpClient}
}
