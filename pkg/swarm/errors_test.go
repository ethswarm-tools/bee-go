package swarm

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestBeeError_UnwrapAndMessage(t *testing.T) {
	inner := errors.New("boom")
	err := WrapBeeError("decode", inner)
	if !errors.Is(err, inner) {
		t.Errorf("errors.Is should find inner")
	}
	if !strings.Contains(err.Error(), "decode") || !strings.Contains(err.Error(), "boom") {
		t.Errorf("error message missing parts: %q", err.Error())
	}
}

func TestBeeArgumentError_AsAndValue(t *testing.T) {
	err := NewBeeArgumentError("invalid depth", 5)
	var arg *BeeArgumentError
	if !errors.As(err, &arg) {
		t.Fatal("errors.As should find argument error")
	}
	if arg.Value != 5 {
		t.Errorf("Value = %v, want 5", arg.Value)
	}
}

func TestNewBeeResponseError_ReadsBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Status:     "500 Internal Server Error",
		Body:       io.NopCloser(strings.NewReader("upstream broke")),
	}
	err := NewBeeResponseError("POST", "http://bee/x", resp)
	if err.Status != 500 {
		t.Errorf("Status = %d", err.Status)
	}
	if !bytes.Equal(err.ResponseBody, []byte("upstream broke")) {
		t.Errorf("ResponseBody = %q", err.ResponseBody)
	}
	if !strings.Contains(err.Error(), "POST http://bee/x") || !strings.Contains(err.Error(), "500") {
		t.Errorf("error string missing parts: %q", err.Error())
	}
	got, ok := IsBeeResponseError(err)
	if !ok || got.Status != 500 {
		t.Errorf("IsBeeResponseError did not match")
	}
}

func TestNewBeeResponseError_BodyCapped(t *testing.T) {
	// Body > 4096 should be truncated.
	huge := strings.Repeat("x", 10000)
	resp := &http.Response{
		StatusCode: 502,
		Status:     "502 Bad Gateway",
		Body:       io.NopCloser(strings.NewReader(huge)),
	}
	err := NewBeeResponseError("GET", "http://bee/y", resp)
	if len(err.ResponseBody) != 4096 {
		t.Errorf("len(ResponseBody) = %d, want 4096", len(err.ResponseBody))
	}
}
