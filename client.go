package bee

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/debug"
	"github.com/ethswarm-tools/bee-go/pkg/file"
	"github.com/ethswarm-tools/bee-go/pkg/gsoc"
	"github.com/ethswarm-tools/bee-go/pkg/postage"
	"github.com/ethswarm-tools/bee-go/pkg/pss"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
	"github.com/gorilla/websocket"
)

// Client is the Bee API client.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	dialer     *websocket.Dialer

	// Services
	API     *api.Service
	Debug   *debug.Service
	File    *file.Service
	Postage *postage.Service
	Swarm   *swarm.Service
	PSS     *pss.Service
	GSOC    *gsoc.Service
}

// ClientOption is a function that configures a Client.
type ClientOption func(*Client)

// WithHTTPClient configures the Client to use the given HTTP client.
func WithHTTPClient(c *http.Client) ClientOption {
	return func(client *Client) {
		client.httpClient = c
	}
}

// NewClient creates a new Bee API client.
func NewClient(rawURL string, opts ...ClientOption) (*Client, error) {
	if !strings.HasSuffix(rawURL, "/") {
		rawURL += "/"
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	c := &Client{
		baseURL:    u,
		httpClient: http.DefaultClient,
		dialer:     websocket.DefaultDialer,
	}

	for _, opt := range opts {
		opt(c)
	}

	// Initialize Services
	c.API = api.NewService(c.baseURL, c.httpClient)
	c.Debug = debug.NewService(c.baseURL, c.httpClient)
	c.File = file.NewService(c.baseURL, c.httpClient)
	c.Postage = postage.NewService(c.baseURL, c.httpClient)
	c.Swarm = swarm.NewService(c.baseURL, c.httpClient)
	c.PSS = pss.NewService(c.baseURL, c.httpClient, c.dialer)
	c.GSOC = gsoc.NewService(c.baseURL, c.httpClient, c.dialer, c.File)

	return c, nil
}
