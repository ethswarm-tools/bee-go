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

// Client is the top-level Bee API client. It bundles one sub-service
// per Bee API domain (see the package doc for the layout); construct it
// once with [NewClient] and reuse — every sub-service shares the same
// underlying *http.Client.
//
// High-level helpers that span multiple sub-services
// ([Client.BuyStorage], [Client.ExtendStorage], [Client.GetStorageCost]
// and friends) live on Client itself.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	dialer     *websocket.Dialer

	// API is the cross-cutting endpoint group: pin, tag, stewardship,
	// grantee, envelope, and "is reference retrievable?" checks.
	API *api.Service
	// Debug is the operator / observability surface: health, versions,
	// peers, accounting, chequebook, stake, transactions, loggers.
	Debug *debug.Service
	// File handles every "data goes in or out of Bee" endpoint: bytes,
	// files, chunks, SOCs, feeds, and tar-packed collections.
	File *file.Service
	// Postage handles postage-batch CRUD; pure stamp math is exposed as
	// free functions in [pkg/postage].
	Postage *postage.Service
	// Swarm groups the offline helpers that don't need a Bee node
	// (e.g. content-addressed chunk construction).
	Swarm *swarm.Service
	// PSS is Postal Service: send / subscribe / receive over the
	// neighborhood-routed PSS layer.
	PSS *pss.Service
	// GSOC is Generic Single-Owner Chunk send / subscribe (built on top
	// of the SOC primitive in [pkg/swarm]).
	GSOC *gsoc.Service
}

// ClientOption configures a [Client] at construction time. Pass any
// number to [NewClient]; options are applied in order before the
// sub-services are wired up, so options that swap out the *http.Client
// or websocket dialer take effect for every sub-service.
type ClientOption func(*Client)

// WithHTTPClient overrides the *http.Client used for every sub-service.
// Useful for sharing a connection pool with surrounding code, setting a
// custom timeout, or installing a transport-level interceptor (auth,
// retries, instrumentation). The default is [http.DefaultClient].
func WithHTTPClient(c *http.Client) ClientOption {
	return func(client *Client) {
		client.httpClient = c
	}
}

// NewClient constructs a [Client] pointing at a Bee node's REST API.
// The url should be the base address (e.g. "http://localhost:1633");
// a trailing slash is appended if missing so relative paths resolve
// correctly. Sub-services are wired up in order so the returned Client
// is ready to use.
//
// Returns an error only if rawURL fails to parse.
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
