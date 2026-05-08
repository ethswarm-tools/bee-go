package swarm

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// RedactURL returns a printable form of u with the query string and
// fragment stripped. Used by error messages and the HTTP logger so
// that sensitive values that callers (or the server) put into the
// query string — the SOC signature, an Act publisher key, an
// auth token mistakenly passed via `?token=` — never end up in logs
// or panics. The path is preserved (the path-segment IDs are
// references / hex-encoded peers and considered public).
func RedactURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	r := *u
	r.RawQuery = ""
	r.Fragment = ""
	return r.String()
}

// redactURLString parses s and returns RedactURL(u). On parse failure
// it falls back to the substring before the first '?' — strictly less
// information than the original.
func redactURLString(s string) string {
	if s == "" {
		return ""
	}
	if u, err := url.Parse(s); err == nil {
		return RedactURL(u)
	}
	if i := indexByte(s, '?'); i >= 0 {
		return s[:i]
	}
	return s
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// BeeError is the base type for all errors surfaced by bee-go. Every error
// returned from a `pkg/...` API method either is, or wraps, a *BeeError. Use
// errors.As(err, &target) to inspect the typed forms below.
//
// Mirrors bee-js BeeError.
type BeeError struct {
	Msg string
	// Inner is wrapped with errors.Unwrap semantics; nil for top-level errors.
	Inner error
}

func (e *BeeError) Error() string {
	if e.Inner != nil {
		return e.Msg + ": " + e.Inner.Error()
	}
	return e.Msg
}

func (e *BeeError) Unwrap() error { return e.Inner }

// NewBeeError builds a BeeError without an underlying cause.
func NewBeeError(msg string) *BeeError { return &BeeError{Msg: msg} }

// WrapBeeError wraps inner with a contextual message.
func WrapBeeError(msg string, inner error) *BeeError { return &BeeError{Msg: msg, Inner: inner} }

// BeeArgumentError indicates the caller passed an invalid argument. Value is
// the offending input (best-effort; nil if not applicable). Mirrors bee-js
// BeeArgumentError.
type BeeArgumentError struct {
	BeeError
	Value any
}

// NewBeeArgumentError builds a BeeArgumentError from a message and value.
func NewBeeArgumentError(msg string, value any) *BeeArgumentError {
	return &BeeArgumentError{BeeError: BeeError{Msg: msg}, Value: value}
}

// BeeResponseError indicates Bee returned a non-2xx status. Method/URL pin
// the failed request; Status/StatusText carry the HTTP outcome; ResponseBody
// is the raw body bytes (read up to a small cap so we don't OOM on huge
// error pages). Mirrors bee-js BeeResponseError.
type BeeResponseError struct {
	BeeError
	Method       string
	URL          string
	Status       int
	StatusText   string
	ResponseBody []byte
}

// NewBeeResponseError reads up to 4 KiB of resp.Body and constructs a typed
// error. The body is consumed but resp.Body is not closed — callers are
// expected to defer Close themselves as is conventional with net/http.
//
// The URL stored on the typed error and embedded in the message has
// the query string stripped via [RedactURL] so that sensitive values
// (signatures, act-publisher keys, mistakenly-passed `?token=`) don't
// end up in logs or wrapping error chains.
func NewBeeResponseError(method, requestURL string, resp *http.Response) *BeeResponseError {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	redacted := redactURLString(requestURL)
	// resp.Status already begins with the numeric code (e.g. "422 Unprocessable
	// Entity"), so we don't repeat resp.StatusCode here.
	msg := fmt.Sprintf("%s %s: %s", method, redacted, resp.Status)
	return &BeeResponseError{
		BeeError:     BeeError{Msg: msg},
		Method:       method,
		URL:          redacted,
		Status:       resp.StatusCode,
		StatusText:   resp.Status,
		ResponseBody: body,
	}
}

// IsBeeResponseError is sugar for errors.As + nil check; returns the typed
// error and true if err contains one.
func IsBeeResponseError(err error) (*BeeResponseError, bool) {
	var target *BeeResponseError
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

// IsBeeArgumentError is the analogous helper for argument errors.
func IsBeeArgumentError(err error) (*BeeArgumentError, bool) {
	var target *BeeArgumentError
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

// CheckResponse returns nil if resp is 2xx, otherwise a *BeeResponseError
// annotated with the request method and URL (read from resp.Request, which
// http.Client.Do populates). Use it as the standard "happy path or typed
// error" check after every Bee call.
func CheckResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	method, reqURL := "", ""
	if resp.Request != nil {
		method = resp.Request.Method
		if resp.Request.URL != nil {
			reqURL = RedactURL(resp.Request.URL)
		}
	}
	return NewBeeResponseError(method, reqURL, resp)
}
