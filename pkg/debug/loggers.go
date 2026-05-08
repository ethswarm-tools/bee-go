package debug

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

// LogLevels lists the verbosity tokens accepted by Bee's
// PUT /loggers/{exp}/{verbosity} route (pkg/api/logger.go on the Bee
// server). Anything outside this set is rejected with a 400; [SetLogger]
// validates client-side before sending.
var LogLevels = []string{"none", "error", "warning", "info", "debug", "all"}

// Logger is one entry in the LoggerResponse.Loggers list.
type Logger struct {
	Logger    string `json:"logger"`
	Verbosity string `json:"verbosity"`
	Subsystem string `json:"subsystem"`
	ID        string `json:"id"`
}

// LoggerResponse is the response from /loggers and /loggers/{exp}: the
// flat logger list plus a tree representation. Tree is intentionally
// raw JSON — the structure is recursive and rarely needed in code.
type LoggerResponse struct {
	Tree    json.RawMessage `json:"tree"`
	Loggers []Logger        `json:"loggers"`
}

// GetLoggers returns every logger registered in the running node.
// Mirrors bee-js Bee.getLoggers.
func (s *Service) GetLoggers(ctx context.Context) (LoggerResponse, error) {
	return s.getLoggers(ctx, "loggers")
}

// GetLoggersByExpression returns loggers matching the given regex /
// subsystem expression. The expression is base64-encoded into the URL
// per the Bee /loggers/{exp} contract. Mirrors bee-js
// Bee.getLoggersByExpression.
func (s *Service) GetLoggersByExpression(ctx context.Context, expression string) (LoggerResponse, error) {
	enc := base64.StdEncoding.EncodeToString([]byte(expression))
	return s.getLoggers(ctx, "loggers/"+enc)
}

func (s *Service) getLoggers(ctx context.Context, path string) (LoggerResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: path})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return LoggerResponse{}, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return LoggerResponse{}, err
	}
	defer resp.Body.Close()
	if err := swarm.CheckResponse(resp); err != nil {
		return LoggerResponse{}, err
	}
	var res LoggerResponse
	if err := swarm.DecodeJSONResponse(resp, &res); err != nil {
		return LoggerResponse{}, err
	}
	return res, nil
}

// SetLogger sets the verbosity of every logger matching expression.
// The expression is base64-encoded into the URL per the Bee
// PUT /loggers/{exp}/{verbosity} contract. verbosity must be one of
// [LogLevels] ("none|error|warning|info|debug|all"); anything else is
// rejected client-side with an Argument error before the request is
// built.
//
// Pass "." as expression to bump every logger at once (Bee treats it
// as a regex match-all). Mirrors bee-rs DebugApi::set_logger and
// bee-py client.debug.set_logger.
func (s *Service) SetLogger(ctx context.Context, expression, verbosity string) error {
	if !slices.Contains(LogLevels, verbosity) {
		return fmt.Errorf("verbosity %q not in %v", verbosity, LogLevels)
	}
	enc := base64.StdEncoding.EncodeToString([]byte(expression))
	u := s.baseURL.ResolveReference(&url.URL{Path: "loggers/" + enc + "/" + verbosity})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return swarm.CheckResponse(resp)
}

// SetLoggerVerbosity is broken — Bee's actual route is
// PUT /loggers/{exp}/{verbosity}; verbosity is mandatory in the path.
// This method emits PUT /loggers/{exp} which 404s on every real Bee
// build. It only ever "succeeded" against mock servers wired to the
// wrong path.
//
// Deprecated: use [Service.SetLogger] instead — it takes both the
// expression and verbosity and emits the correct path. Kept for
// backwards compatibility; returns an error explaining the breakage.
func (s *Service) SetLoggerVerbosity(ctx context.Context, expression string) error {
	_ = expression
	_ = ctx
	return fmt.Errorf("SetLoggerVerbosity is broken — Bee's route requires a verbosity component. " +
		"Call SetLogger(ctx, expression, verbosity) instead")
}
