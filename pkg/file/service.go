package file

import (
	"net/http"
	"net/url"
)

// Service handles file operations.
type Service struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewService creates a new file service.
func NewService(baseURL *url.URL, httpClient *http.Client) *Service {
	return &Service{baseURL: baseURL, httpClient: httpClient}
}
