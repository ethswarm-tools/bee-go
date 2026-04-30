package debug

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/ethersphere/bee-go/pkg/swarm"
)

// Health checks if the Bee node is healthy.
func (s *Service) Health(ctx context.Context) (bool, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "health"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return false, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return false, err
	}
	return true, nil
}

// Readiness checks if the Bee node is ready to serve requests.
func (s *Service) Readiness(ctx context.Context) (bool, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "readiness"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return false, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// StatusResponse represents the node status.
type StatusResponse struct {
	Overlay                 string  `json:"overlay"`
	Proximity               int     `json:"proximity"`
	BeeMode                 string  `json:"beeMode"`
	ReserveSize             int64   `json:"reserveSize"`
	ReserveSizeWithinRadius int64   `json:"reserveSizeWithinRadius"`
	PullsyncRate            float64 `json:"pullsyncRate"`
	StorageRadius           int     `json:"storageRadius"`
	ConnectedPeers          int     `json:"connectedPeers"`
	NeighborhoodSize        int     `json:"neighborhoodSize"`
	BatchCommitment         int64   `json:"batchCommitment"`
	IsReachable             bool    `json:"isReachable"`
	LastSyncedBlock         int64   `json:"lastSyncedBlock"`
	CommittedDepth          int     `json:"committedDepth"`
	IsWarmingUp             bool    `json:"isWarmingUp"`
}

// Status checks the status of the Bee node components.
func (s *Service) Status(ctx context.Context) (StatusResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "status"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return StatusResponse{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return StatusResponse{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return StatusResponse{}, err
	}

	var res StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return StatusResponse{}, err
	}

	return res, nil
}

// PeerStatus is one entry in the StatusPeers response. It mirrors
// StatusResponse but adds RequestFailed (set when Bee couldn't reach
// the peer in time and the rest of the snapshot is zero-valued).
type PeerStatus struct {
	StatusResponse
	RequestFailed bool `json:"requestFailed,omitempty"`
}

// StatusPeers returns the status snapshot of every currently connected
// peer (bee mode, reserve size, sync state, …). Snapshots are gathered
// in parallel by the Bee node with a 10-second per-peer timeout; peers
// that don't respond have RequestFailed=true.
//
// Bee node endpoint: GET /status/peers. Not exposed by bee-js.
func (s *Service) StatusPeers(ctx context.Context) ([]PeerStatus, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "status/peers"})
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
		Snapshots []PeerStatus `json:"snapshots"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Snapshots, nil
}

// Neighborhood describes one neighborhood the node knows about: its
// binary prefix string, the reserve size within radius, and the
// proximity order.
type Neighborhood struct {
	Neighborhood            string `json:"neighborhood"`
	ReserveSizeWithinRadius int    `json:"reserveSizeWithinRadius"`
	Proximity               uint8  `json:"proximity"`
}

// StatusNeighborhoods returns reserve statistics per neighborhood.
// Useful for storage-layer diagnostics.
//
// Bee node endpoint: GET /status/neighborhoods. Not exposed by bee-js.
func (s *Service) StatusNeighborhoods(ctx context.Context) ([]Neighborhood, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "status/neighborhoods"})
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
		Neighborhoods []Neighborhood `json:"neighborhoods"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Neighborhoods, nil
}

// GetWelcomeMessage returns the P2P welcome message the node greets new
// peers with. Mirrors bee-js Bee.getWelcomeMessage.
func (s *Service) GetWelcomeMessage(ctx context.Context) (string, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "welcome-message"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
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
		WelcomeMessage string `json:"welcomeMessage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.WelcomeMessage, nil
}

// SetWelcomeMessage updates the P2P welcome message. Mirrors bee-js
// Bee.setWelcomeMessage.
func (s *Service) SetWelcomeMessage(ctx context.Context, message string) error {
	u := s.baseURL.ResolveReference(&url.URL{Path: "welcome-message"})
	body, err := json.Marshal(struct {
		WelcomeMessage string `json:"welcomeMessage"`
	}{WelcomeMessage: message})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return swarm.CheckResponse(resp)
}

// NodeInfo represents the Bee node configuration.
type NodeInfo struct {
	BeeMode           string `json:"beeMode"`
	ChequebookEnabled bool   `json:"chequebookEnabled"`
	SwapEnabled       bool   `json:"swapEnabled"`
}

// NodeInfo returns information about the Bee node configuration.
func (s *Service) NodeInfo(ctx context.Context) (*NodeInfo, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "node"})
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

	var info NodeInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

// ChainStateResponse represents the chain state.
type ChainStateResponse struct {
	ChainTip     uint64 `json:"chainTip"`
	Block        uint64 `json:"block"`
	CurrentPrice uint64 `json:"currentPrice"`
}

// ChainState retrieves the current chain state.
func (s *Service) ChainState(ctx context.Context) (ChainStateResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "chainstate"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return ChainStateResponse{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return ChainStateResponse{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return ChainStateResponse{}, err
	}

	var res ChainStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return ChainStateResponse{}, err
	}
	return res, nil
}

// ReserveStateResponse represents the reserve state.
type ReserveStateResponse struct {
	Radius        uint8 `json:"radius"`
	StorageRadius uint8 `json:"storageRadius"`
	Commitment    int64 `json:"commitment"`
}

// ReserveState retrieves the reserve state.
func (s *Service) ReserveState(ctx context.Context) (ReserveStateResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "reservestate"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return ReserveStateResponse{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return ReserveStateResponse{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return ReserveStateResponse{}, err
	}

	var res ReserveStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return ReserveStateResponse{}, err
	}
	return res, nil
}

// TopologyResponse represents the node topology.
type TopologyResponse struct {
	BaseAddr       string `json:"baseAddr"`
	Population     int    `json:"population"`
	Connected      int    `json:"connected"`
	Timestamp      string `json:"timestamp"`
	NnLowWatermark int    `json:"nnLowWatermark"`
	Depth          uint8  `json:"depth"`
}

// Topology retrieves the node topology.
func (s *Service) Topology(ctx context.Context) (TopologyResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "topology"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return TopologyResponse{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return TopologyResponse{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return TopologyResponse{}, err
	}

	var res TopologyResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return TopologyResponse{}, err
	}
	return res, nil
}

// Peer represents a connected peer.
type Peer struct {
	Address  string `json:"address"`
	FullNode bool   `json:"fullNode"`
}

// PeersResponse represents the list of peers.
type PeersResponse struct {
	Peers []Peer `json:"peers"`
}

// Peers retrieves the list of connected peers.
func (s *Service) Peers(ctx context.Context) (PeersResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "peers"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return PeersResponse{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return PeersResponse{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return PeersResponse{}, err
	}

	var res PeersResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return PeersResponse{}, err
	}
	return res, nil
}

// AddressesResponse represents the node's addresses.
type AddressesResponse struct {
	Overlay      string   `json:"overlay"`
	Underlay     []string `json:"underlay"`
	Ethereum     string   `json:"ethereum"`
	PublicKey    string   `json:"publicKey"`
	PssPublicKey string   `json:"pssPublicKey"`
}

// Addresses retrieves the node's addresses.
func (s *Service) Addresses(ctx context.Context) (AddressesResponse, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "addresses"})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return AddressesResponse{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return AddressesResponse{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return AddressesResponse{}, err
	}

	var res AddressesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return AddressesResponse{}, err
	}
	return res, nil
}
