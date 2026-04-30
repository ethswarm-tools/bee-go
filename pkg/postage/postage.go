package postage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"

	"github.com/ethersphere/bee-go/pkg/swarm"
)

// PostageBatch represents a Swarm postage batch.
type PostageBatch struct {
	BatchID     swarm.BatchID `json:"batchID"`
	Value       *big.Int      `json:"-"`
	Start       uint64        `json:"start"`
	Owner       string        `json:"owner"`
	Depth       uint8         `json:"depth"`
	BucketDepth uint8         `json:"bucketDepth"`
	Immutable   bool          `json:"immutable"`
	BatchTTL    int64         `json:"batchTTL"`
	Utilization uint32        `json:"utilization"`
	Usable      bool          `json:"usable"`
	Label       string        `json:"label"`
	BlockNumber uint64        `json:"blockNumber"`
}

type postageBatchJSON struct {
	BatchID     string `json:"batchID"`
	Value       string `json:"value"`
	Start       uint64 `json:"start"`
	Owner       string `json:"owner"`
	Depth       uint8  `json:"depth"`
	BucketDepth uint8  `json:"bucketDepth"`
	Immutable   bool   `json:"immutable"`
	BatchTTL    int64  `json:"batchTTL"`
	Utilization uint32 `json:"utilization"`
	Usable      bool   `json:"usable"`
	Label       string `json:"label"`
	BlockNumber uint64 `json:"blockNumber"`
}

// UnmarshalJSON implements custom unmarshalling for PostageBatch.
func (pb *PostageBatch) UnmarshalJSON(b []byte) error {
	var v postageBatchJSON
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	id, err := swarm.BatchIDFromHex(v.BatchID)
	if err != nil {
		return fmt.Errorf("invalid batchID: %w", err)
	}
	pb.BatchID = id
	pb.Start = v.Start
	pb.Owner = v.Owner
	pb.Depth = v.Depth
	pb.BucketDepth = v.BucketDepth
	pb.Immutable = v.Immutable
	pb.BatchTTL = v.BatchTTL
	pb.Utilization = v.Utilization
	pb.Usable = v.Usable
	pb.Label = v.Label
	pb.BlockNumber = v.BlockNumber

	if v.Value != "" {
		val := new(big.Int)
		if _, ok := val.SetString(v.Value, 10); !ok {
			return fmt.Errorf("invalid big.Int string: %s", v.Value)
		}
		pb.Value = val
	}
	return nil
}

// GetPostageBatches retrieves all postage batches.
func (s *Service) GetPostageBatches(ctx context.Context) ([]PostageBatch, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "batches"})
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
		Batches []PostageBatch `json:"batches"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Batches, nil
}

// CreatePostageBatch purchases a new postage batch.
// amount: initial balance per chunk (big.Int)
// depth: batch depth (uint8) 17 -> 2^17 chunks
// label: optional label for the batch
func (s *Service) CreatePostageBatch(ctx context.Context, amount *big.Int, depth uint8, label string) (swarm.BatchID, error) {
	path := fmt.Sprintf("stamps/%s/%d", amount.String(), depth)
	u := s.baseURL.ResolveReference(&url.URL{Path: path})
	q := u.Query()
	if label != "" {
		q.Set("label", label)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return swarm.BatchID{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return swarm.BatchID{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return swarm.BatchID{}, err
	}

	var res struct {
		BatchID string `json:"batchID"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return swarm.BatchID{}, err
	}
	return swarm.BatchIDFromHex(res.BatchID)
}

// TopUpBatch adds more value (amount) to an existing batch.
func (s *Service) TopUpBatch(ctx context.Context, batchID swarm.BatchID, amount *big.Int) error {
	path := fmt.Sprintf("stamps/topup/%s/%s", batchID.Hex(), amount.String())
	u := s.baseURL.ResolveReference(&url.URL{Path: path})

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u.String(), nil)
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

// DiluteBatch increases the depth of an existing batch.
func (s *Service) DiluteBatch(ctx context.Context, batchID swarm.BatchID, depth uint8) error {
	path := fmt.Sprintf("stamps/dilute/%s/%d", batchID.Hex(), depth)
	u := s.baseURL.ResolveReference(&url.URL{Path: path})

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u.String(), nil)
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

// GetPostageBatch retrieves a single postage batch by ID.
func (s *Service) GetPostageBatch(ctx context.Context, batchID swarm.BatchID) (*PostageBatch, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("stamps/%s", batchID.Hex())})
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

	var pb PostageBatch
	if err := json.NewDecoder(resp.Body).Decode(&pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

// Stamp Math Utilities

// GetStampUsage calculates the fractional usage [0,1] of a postage batch.
func GetStampUsage(utilization uint32, depth uint8, bucketDepth uint8) float64 {
	denominator := 1 << (depth - bucketDepth)
	return float64(utilization) / float64(denominator)
}

// GetStampTheoreticalBytes is the upper bound for a batch of the given
// depth: 4096 bytes per chunk × 2^depth chunks.
func GetStampTheoreticalBytes(depth int) int64 {
	return 4096 * (1 << int64(depth))
}

// GetStampCost returns 2^depth × amount, the BZZ-PLUR cost of buying a
// batch of the given depth and amount.
func GetStampCost(depth int, amount *big.Int) *big.Int {
	pow := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(depth)), nil)
	return new(big.Int).Mul(pow, amount)
}

// effectiveSizeBreakpoints holds (depth, GB) entries from the Swarm
// effective-utilisation table for encrypted, medium-erasure batches.
// Values mirror bee-js stamps.ts.
var effectiveSizeBreakpoints = [...]struct {
	depth int
	gb    float64
}{
	{17, 0.00004089},
	{18, 0.00609},
	{19, 0.10249},
	{20, 0.62891},
	{21, 2.38},
	{22, 7.07},
	{23, 18.24},
	{24, 43.04},
	{25, 96.5},
	{26, 208.52},
	{27, 435.98},
	{28, 908.81},
	{29, 1870},
	{30, 3810},
	{31, 7730},
	{32, 15610},
	{33, 31430},
	{34, 63150},
}

// GetStampEffectiveBytes returns the practical capacity for the given
// depth, using the Swarm effective-utilisation table for depths 17–34
// and the 0.9 max-utilization approximation outside that range.
func GetStampEffectiveBytes(depth int) int64 {
	if depth < 17 {
		return 0
	}
	for _, e := range effectiveSizeBreakpoints {
		if e.depth == depth {
			return int64(e.gb * 1_000_000_000)
		}
	}
	return int64(float64(GetStampTheoreticalBytes(depth)) * 0.9)
}

// GetStampDuration estimates the TTL of a batch given its `amount`,
// `pricePerBlock` (PLUR/block) and `blockTime` (seconds/block):
//
//	seconds = amount * blockTime / pricePerBlock
//
// pricePerBlock and blockTime should come from the Bee node where
// possible (chainstate). Mirrors bee-js getStampDuration.
func GetStampDuration(amount *big.Int, pricePerBlock uint64, blockTime uint64) swarm.Duration {
	if pricePerBlock == 0 {
		return swarm.ZeroDuration
	}
	num := new(big.Int).Mul(amount, new(big.Int).SetUint64(blockTime))
	seconds := new(big.Int).Quo(num, new(big.Int).SetUint64(pricePerBlock))
	return swarm.DurationFromSeconds(float64(seconds.Int64()))
}

// GetAmountForDuration computes the `amount` (PLUR per chunk) needed to
// fund a batch for the given duration:
//
//	amount = (duration / blockTime) * pricePerBlock + 1
//
// The `+ 1` matches bee-js to compensate for integer division and avoid
// short funding.
func GetAmountForDuration(duration swarm.Duration, pricePerBlock uint64, blockTime uint64) *big.Int {
	if blockTime == 0 {
		return big.NewInt(1)
	}
	blocks := new(big.Int).Quo(big.NewInt(duration.ToSeconds()), new(big.Int).SetUint64(blockTime))
	return new(big.Int).Add(new(big.Int).Mul(blocks, new(big.Int).SetUint64(pricePerBlock)), big.NewInt(1))
}

// GetDepthForSize returns the smallest depth whose effective capacity
// covers the given size. Falls back to 35 for sizes larger than the
// table covers.
func GetDepthForSize(size swarm.Size) int {
	bytes := size.ToBytes()
	for _, e := range effectiveSizeBreakpoints {
		if bytes <= int64(e.gb*1_000_000_000) {
			return e.depth
		}
	}
	return 35
}
