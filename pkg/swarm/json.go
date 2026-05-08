package swarm

import (
	"encoding/json"
	"net/http"
)

// MaxJSONResponseBytes is the default cap applied by [DecodeJSONResponse]
// to protect against a misbehaving or compromised Bee streaming an
// unbounded JSON document and exhausting the client's memory. 32 MiB
// is generous for any structured Bee response — every shipped
// /chequebook, /stamps, /peers, /status, /accounting, /loggers,
// /reservestate response is well under one MiB in practice.
const MaxJSONResponseBytes = 32 << 20

// DecodeJSONResponse decodes resp.Body into v with the body wrapped
// in [http.MaxBytesReader] so a malicious or runaway server cannot
// OOM the client. Callers remain responsible for resp.Body.Close.
//
// Use this helper instead of `json.NewDecoder(resp.Body).Decode(&v)`
// for any endpoint that returns structured JSON.
func DecodeJSONResponse(resp *http.Response, v any) error {
	r := http.MaxBytesReader(nil, resp.Body, MaxJSONResponseBytes)
	return json.NewDecoder(r).Decode(v)
}
