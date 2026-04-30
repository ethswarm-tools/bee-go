package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

// GetBlocklist retrieves the list of blocklisted peers.
func (s *Service) GetBlocklist(ctx context.Context) ([]Peer, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "blocklist"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return nil, err
	}

	var res struct {
		Peers []Peer `json:"peers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Peers, nil
}

// RemovePeer removes a peer from the node.
func (s *Service) RemovePeer(ctx context.Context, address string) error {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("peers/%s", address)})
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return err
	}
	return nil
}

// PingPeer pings a peer.
func (s *Service) PingPeer(ctx context.Context, address string) (*string, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("pingpong/%s", address)})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return nil, err
	}

	var res struct {
		RTT string `json:"rtt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res.RTT, nil
}

// ConnectPeer manually dials a peer at the given multiaddress (e.g.
// "/dns/bee.example.com/tcp/1634/p2p/16Uiu2HAm…"). Returns the resulting
// overlay address. The leading "/" is added by Bee, so callers may pass
// the multiaddress with or without it.
//
// Bee node endpoint: POST /connect/{multi-address}. Not exposed by bee-js.
func (s *Service) ConnectPeer(ctx context.Context, multiaddr string) (string, error) {
	// The Bee router escapes the multiaddress as a path segment but
	// re-adds a leading slash internally. We strip a leading slash so
	// double slashes aren't introduced into the URL path.
	for len(multiaddr) > 0 && multiaddr[0] == '/' {
		multiaddr = multiaddr[1:]
	}
	u := s.baseURL.ResolveReference(&url.URL{Path: "connect/" + multiaddr})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if err := swarm.CheckResponse(resp); err != nil {
		return "", err
	}
	var res struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.Address, nil
}
