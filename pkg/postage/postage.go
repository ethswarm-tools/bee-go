package postage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
)

// PostageBatch represents a Swarm postage batch.
// PostageBatch represents a Swarm postage batch.
type PostageBatch struct {
	BatchID     string   `json:"batchID"`
	Value       *big.Int `json:"-"`
	Start       uint64   `json:"start"`
	Owner       string   `json:"owner"`
	Depth       uint8    `json:"depth"`
	BucketDepth uint8    `json:"bucketDepth"`
	Immutable   bool     `json:"immutable"`
	BatchTTL    int64    `json:"batchTTL"`
	Utilization uint32   `json:"utilization"`
	Usable      bool     `json:"usable"`
	Label       string   `json:"label"`
	BlockNumber uint64   `json:"blockNumber"`
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
	pb.BatchID = v.BatchID
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get batches failed with status: %d", resp.StatusCode)
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
func (s *Service) CreatePostageBatch(ctx context.Context, amount *big.Int, depth uint8, label string) (string, error) {
	path := fmt.Sprintf("stamps/%s/%d", amount.String(), depth)
	u := s.baseURL.ResolveReference(&url.URL{Path: path})
	q := u.Query()
	if label != "" {
		q.Set("label", label)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create batch failed with status: %d", resp.StatusCode)
	}

	var res struct {
		BatchID string `json:"batchID"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.BatchID, nil
}

// TopUpBatch adds more value (amount) to an existing batch.
func (s *Service) TopUpBatch(ctx context.Context, batchID string, amount *big.Int) error {
	path := fmt.Sprintf("stamps/topup/%s/%s", batchID, amount.String())
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("topup batch failed with status: %d", resp.StatusCode)
	}
	return nil
}

// DiluteBatch increases the depth of an existing batch.
func (s *Service) DiluteBatch(ctx context.Context, batchID string, depth uint8) error {
	path := fmt.Sprintf("stamps/dilute/%s/%d", batchID, depth)
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("dilute batch failed with status: %d", resp.StatusCode)
	}
	return nil
}

// GetPostageBatch retrieves a single postage batch by ID.
func (s *Service) GetPostageBatch(ctx context.Context, batchID string) (*PostageBatch, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("stamps/%s", batchID)})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get batch failed with status: %d", resp.StatusCode)
	}

	var pb PostageBatch
	if err := json.NewDecoder(resp.Body).Decode(&pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

// Stamp Math Utilities

// GetStampUsage calculates usage of postage batch.
func GetStampUsage(utilization uint32, depth uint8, bucketDepth uint8) float64 {
	denominator := 1 << (depth - bucketDepth)
	return float64(utilization) / float64(denominator)
}

// GetStampTheoreticalBytes calculates theoretical max size.
func GetStampTheoreticalBytes(depth int) int64 {
	return 4096 * (1 << int64(depth))
}

// GetStampCost calculates cost.
func GetStampCost(depth int, amount *big.Int) *big.Int {
	// 2^depth * amount
	pow := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(depth)), nil)
	return new(big.Int).Mul(pow, amount)
}

// GetStampEffectiveBytes calculates effective size.
// Simplified version without redundancy/encryption tables for now.
func GetStampEffectiveBytes(depth int) int64 {
	if depth < 17 {
		return 0
	}
	// Using max utilization 0.9 approximation from bee-js fallback
	return int64(float64(GetStampTheoreticalBytes(depth)) * 0.9)
}
