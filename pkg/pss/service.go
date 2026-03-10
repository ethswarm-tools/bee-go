package pss

import (
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

// Service handles PSS operations.
type Service struct {
	baseURL    *url.URL
	httpClient *http.Client
	dialer     *websocket.Dialer
}

// NewService creates a new PSS service.
func NewService(baseURL *url.URL, httpClient *http.Client, dialer *websocket.Dialer) *Service {
	return &Service{baseURL: baseURL, httpClient: httpClient, dialer: dialer}
}
