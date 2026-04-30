package bee

// DevClient is a Bee client variant that talks to a Bee node running in
// "dev" mode (`bee dev`). It wraps the regular Client but documents the
// reduced surface — most chain-state, chequebook, settlement, postage
// purchase and stake endpoints are not available on dev nodes and will
// return a *BeeResponseError with status 404 if called.
//
// Dev-mode endpoints that work today (and are exercised by the existing
// pkg/debug + pkg/file + pkg/api services on this client):
//
//   - Addresses / Topology / NodeInfo / Status (with dev-shaped, simpler
//     JSON — the existing parsers tolerate the missing fields)
//   - Health, Readiness
//   - File upload/download (/bytes, /bzz, /chunks, /soc, /feeds)
//   - PSS subscribe/send
//   - GSOC subscribe/send
//   - Tags, Pins, Stewardship, Grantees
//   - The /stamps endpoints behave as no-ops in dev mode but do not 404
//
// Mirrors bee-js BeeDev. There is no separate Go type because the wire
// shape is a strict subset; using DevClient is purely a signal to the
// reader (and to future helpers that may want to short-circuit chain
// calls).
type DevClient struct {
	*Client
}

// NewDevClient is the dev-mode equivalent of NewClient. Use against a
// `bee dev` node.
func NewDevClient(rawURL string, opts ...ClientOption) (*DevClient, error) {
	c, err := NewClient(rawURL, opts...)
	if err != nil {
		return nil, err
	}
	return &DevClient{Client: c}, nil
}
