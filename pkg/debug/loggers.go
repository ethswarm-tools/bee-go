package debug

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

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
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return LoggerResponse{}, err
	}
	return res, nil
}

// SetLoggerVerbosity sets verbosity for loggers matching expression.
// The expression is base64-encoded into the URL. Mirrors bee-js
// Bee.setLoggerVerbosity.
func (s *Service) SetLoggerVerbosity(ctx context.Context, expression string) error {
	enc := base64.StdEncoding.EncodeToString([]byte(expression))
	u := s.baseURL.ResolveReference(&url.URL{Path: "loggers/" + enc})
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
