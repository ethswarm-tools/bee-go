package bee

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

// HTTPLogger is the [*slog.Logger] used by the request-logging
// http.RoundTripper that the default client installs (see [NewClient]).
// It defaults to [slog.Default] with a "bee.http" group, so logs are
// silent unless the program configures a slog handler at debug level.
//
// To redirect bee-go HTTP logs to a custom logger:
//
//	bee.HTTPLogger = slog.New(myHandler).WithGroup("bee.http")
//
// Mirrors the bee-py "bee.http" logger.
var HTTPLogger = slog.Default().WithGroup("bee.http")

// loggingTransport wraps base and emits one slog record per round-trip
// — debug for successful responses, error for transport failures —
// with method/url/status/elapsed_ms attributes. The wrapper does not
// touch the request body or response body.
type loggingTransport struct {
	base http.RoundTripper
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.base.RoundTrip(req)
	elapsed := time.Since(start)

	// Log the URL with query string and fragment stripped. Bee uses the
	// query for SOC signatures and Act publisher keys; callers may also
	// (incorrectly but defensively guarded against) put auth tokens
	// there. The path itself is hex/identifier-only and considered
	// public.
	redacted := swarm.RedactURL(req.URL)

	if err != nil {
		HTTPLogger.LogAttrs(req.Context(), slog.LevelError,
			"http request failed",
			slog.String("method", req.Method),
			slog.String("url", redacted),
			slog.Int64("elapsed_ms", elapsed.Milliseconds()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}
	HTTPLogger.LogAttrs(req.Context(), slog.LevelDebug,
		"http request",
		slog.String("method", req.Method),
		slog.String("url", redacted),
		slog.Int("status", resp.StatusCode),
		slog.Int64("elapsed_ms", elapsed.Milliseconds()),
	)
	return resp, nil
}
