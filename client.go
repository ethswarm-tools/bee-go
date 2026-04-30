package bee

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/ethersphere/bee-go/pkg/api"
	"github.com/ethersphere/bee-go/pkg/debug"
	"github.com/ethersphere/bee-go/pkg/file"
	"github.com/ethersphere/bee-go/pkg/gsoc"
	"github.com/ethersphere/bee-go/pkg/postage"
	"github.com/ethersphere/bee-go/pkg/pss"
	"github.com/ethersphere/bee-go/pkg/swarm"
	"github.com/gorilla/websocket"
)

// Client is the Bee API client.
type Client struct {
	baseUrl    *url.URL
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
func NewClient(rawUrl string, opts ...ClientOption) (*Client, error) {
	if !strings.HasSuffix(rawUrl, "/") {
		rawUrl += "/"
	}
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	c := &Client{
		baseUrl:    u,
		httpClient: http.DefaultClient,
		dialer:     websocket.DefaultDialer,
	}

	for _, opt := range opts {
		opt(c)
	}

	// Initialize Services
	c.API = api.NewService(c.baseUrl, c.httpClient)
	c.Debug = debug.NewService(c.baseUrl, c.httpClient)
	c.File = file.NewService(c.baseUrl, c.httpClient)
	c.Postage = postage.NewService(c.baseUrl, c.httpClient)
	c.Swarm = swarm.NewService(c.baseUrl, c.httpClient)
	c.PSS = pss.NewService(c.baseUrl, c.httpClient, c.dialer)
	c.GSOC = gsoc.NewService(c.baseUrl, c.httpClient, c.dialer, c.File)

	return c, nil
}
