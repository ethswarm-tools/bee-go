package bee

import (
	"context"
	"math/big"

	"github.com/ethersphere/bee-go/pkg/api"
	"github.com/ethersphere/bee-go/pkg/postage"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

// Network identifies the chain the Bee node runs on. Used to pick the
// block time when computing TTL math (Gnosis = 5s, Ethereum mainnet =
// 12s historically — bee-js defaults non-gnosis to 15s, which we mirror).
type Network int

const (
	// NetworkGnosis is the Gnosis chain (5-second blocks). The default
	// for production Bee deployments today.
	NetworkGnosis Network = iota
	// NetworkMainnet is Ethereum mainnet (15-second blocks per bee-js).
	NetworkMainnet
)

// BlockTimeSeconds returns the per-block time used for stamp TTL math.
func (n Network) BlockTimeSeconds() uint64 {
	if n == NetworkMainnet {
		return 15
	}
	return 5
}

// StorageOptions configures BuyStorage / ExtendStorage. Encryption and
// RedundancyLevel slots are placeholders for future capacity-table
// lookups; today they are accepted but unused. Network selects block
// time for amount-from-duration math.
//
// Mirrors bee-js's encryption / erasureCodeLevel / network fan-out across
// the storage methods.
type StorageOptions struct {
	Network         Network
	PostageOptions  *api.PostageBatchOptions
	Encryption      bool
	RedundancyLevel api.RedundancyLevel
}

func (o *StorageOptions) network() Network {
	if o == nil {
		return NetworkGnosis
	}
	return o.Network
}

func (o *StorageOptions) postageOpts() *api.PostageBatchOptions {
	if o == nil {
		return nil
	}
	return o.PostageOptions
}

// chainPrice fetches the current price-per-block from the Bee node via
// /chainstate. We hit it through Debug because that's where ChainState
// lives in the Go client today.
func (c *Client) chainPrice(ctx context.Context) (uint64, error) {
	st, err := c.Debug.ChainState(ctx)
	if err != nil {
		return 0, err
	}
	return st.CurrentPrice, nil
}

// BuyStorage creates a postage batch sized + funded for the given size
// and duration. Equivalent to bee-js Bee.buyStorage: compute depth from
// size, amount from duration + chainstate price, then CreatePostageBatch.
func (c *Client) BuyStorage(ctx context.Context, size swarm.Size, duration swarm.Duration, opts *StorageOptions) (swarm.BatchID, error) {
	price, err := c.chainPrice(ctx)
	if err != nil {
		return swarm.BatchID{}, err
	}
	blockTime := opts.network().BlockTimeSeconds()
	amount := postage.GetAmountForDuration(duration, price, blockTime)
	depth := postage.GetDepthForSize(size)
	return c.Postage.CreatePostageBatch(ctx, amount, uint8(depth), labelFromOptions(opts.postageOpts()))
}

// GetStorageCost returns the BZZ cost of buying a batch sized for `size`
// and lasting `duration`. Mirrors bee-js Bee.getStorageCost.
func (c *Client) GetStorageCost(ctx context.Context, size swarm.Size, duration swarm.Duration, opts *StorageOptions) (swarm.BZZ, error) {
	price, err := c.chainPrice(ctx)
	if err != nil {
		return swarm.BZZ{}, err
	}
	blockTime := opts.network().BlockTimeSeconds()
	amount := postage.GetAmountForDuration(duration, price, blockTime)
	depth := postage.GetDepthForSize(size)
	return swarm.NewBZZ(postage.GetStampCost(depth, amount)), nil
}

// ExtendStorage extends a batch's size (absolute) and duration
// (relative). Tops up to fund the new amount and dilutes if the new
// depth is greater. Mirrors bee-js Bee.extendStorage.
func (c *Client) ExtendStorage(ctx context.Context, batchID swarm.BatchID, size swarm.Size, duration swarm.Duration, opts *StorageOptions) error {
	batch, err := c.Postage.GetPostageBatch(ctx, batchID)
	if err != nil {
		return err
	}
	price, err := c.chainPrice(ctx)
	if err != nil {
		return err
	}
	blockTime := opts.network().BlockTimeSeconds()
	depth := postage.GetDepthForSize(size)
	depthDelta := depth - int(batch.Depth)
	multiplier := big.NewInt(1)
	if depthDelta > 0 {
		multiplier = new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(depthDelta)), nil)
	}
	additionalAmount := postage.GetAmountForDuration(duration, price, blockTime)
	currentAmount := postage.GetAmountForDuration(swarm.DurationFromSeconds(float64(batch.BatchTTL)), price, blockTime)

	var targetAmount *big.Int
	if duration.IsZero() {
		targetAmount = new(big.Int).Mul(currentAmount, multiplier)
	} else {
		targetAmount = new(big.Int).Mul(new(big.Int).Add(currentAmount, additionalAmount), multiplier)
	}
	amountDelta := new(big.Int).Sub(targetAmount, currentAmount)

	if amountDelta.Sign() > 0 {
		if err := c.Postage.TopUpBatch(ctx, batchID, amountDelta); err != nil {
			return err
		}
	}
	if depthDelta > 0 {
		return c.Postage.DiluteBatch(ctx, batchID, uint8(depth))
	}
	return nil
}

// ExtendStorageSize extends a batch's depth to cover `size`. Errors if
// the new depth is not strictly greater than the current depth.
func (c *Client) ExtendStorageSize(ctx context.Context, batchID swarm.BatchID, size swarm.Size, opts *StorageOptions) error {
	batch, err := c.Postage.GetPostageBatch(ctx, batchID)
	if err != nil {
		return err
	}
	price, err := c.chainPrice(ctx)
	if err != nil {
		return err
	}
	blockTime := opts.network().BlockTimeSeconds()
	depth := postage.GetDepthForSize(size)
	delta := depth - int(batch.Depth)
	if delta <= 0 {
		return swarm.NewBeeArgumentError("new depth must be greater than current", depth)
	}
	currentAmount := postage.GetAmountForDuration(swarm.DurationFromSeconds(float64(batch.BatchTTL)), price, blockTime)
	// (currentAmount * (2^delta - 1)) + 1
	mul := new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(delta)), nil), big.NewInt(1))
	topup := new(big.Int).Add(new(big.Int).Mul(currentAmount, mul), big.NewInt(1))
	if err := c.Postage.TopUpBatch(ctx, batchID, topup); err != nil {
		return err
	}
	return c.Postage.DiluteBatch(ctx, batchID, uint8(depth))
}

// ExtendStorageDuration tops up the batch by enough to extend its TTL by
// `duration`. Mirrors bee-js Bee.extendStorageDuration.
func (c *Client) ExtendStorageDuration(ctx context.Context, batchID swarm.BatchID, duration swarm.Duration, opts *StorageOptions) error {
	price, err := c.chainPrice(ctx)
	if err != nil {
		return err
	}
	amount := postage.GetAmountForDuration(duration, opts.network().BlockTimeSeconds(), price)
	return c.Postage.TopUpBatch(ctx, batchID, amount)
}

// GetExtensionCost returns the cost of extending the batch by both size
// and duration. Mirrors bee-js Bee.getExtensionCost.
func (c *Client) GetExtensionCost(ctx context.Context, batchID swarm.BatchID, size swarm.Size, duration swarm.Duration, opts *StorageOptions) (swarm.BZZ, error) {
	batch, err := c.Postage.GetPostageBatch(ctx, batchID)
	if err != nil {
		return swarm.BZZ{}, err
	}
	price, err := c.chainPrice(ctx)
	if err != nil {
		return swarm.BZZ{}, err
	}
	blockTime := opts.network().BlockTimeSeconds()
	var amount *big.Int
	if duration.IsZero() {
		amount = big.NewInt(0)
	} else {
		amount = postage.GetAmountForDuration(duration, price, blockTime)
	}
	depth := postage.GetDepthForSize(size)
	currentAmount := postage.GetAmountForDuration(swarm.DurationFromSeconds(float64(batch.BatchTTL)), price, blockTime)
	currentCost := postage.GetStampCost(int(batch.Depth), currentAmount)
	maxDepth := depth
	if int(batch.Depth) > maxDepth {
		maxDepth = int(batch.Depth)
	}
	newCost := postage.GetStampCost(maxDepth, new(big.Int).Add(currentAmount, amount))
	return swarm.NewBZZ(new(big.Int).Sub(newCost, currentCost)), nil
}

// GetSizeExtensionCost returns the BZZ cost of extending the batch's
// depth (absolute) to cover `size`. Errors if the requested depth is
// not greater than the current depth.
func (c *Client) GetSizeExtensionCost(ctx context.Context, batchID swarm.BatchID, size swarm.Size, opts *StorageOptions) (swarm.BZZ, error) {
	batch, err := c.Postage.GetPostageBatch(ctx, batchID)
	if err != nil {
		return swarm.BZZ{}, err
	}
	price, err := c.chainPrice(ctx)
	if err != nil {
		return swarm.BZZ{}, err
	}
	blockTime := opts.network().BlockTimeSeconds()
	depth := postage.GetDepthForSize(size)
	if depth <= int(batch.Depth) {
		return swarm.BZZ{}, swarm.NewBeeArgumentError("new depth must be greater than current", depth)
	}
	currentAmount := postage.GetAmountForDuration(swarm.DurationFromSeconds(float64(batch.BatchTTL)), price, blockTime)
	currentCost := postage.GetStampCost(int(batch.Depth), currentAmount)
	newCost := postage.GetStampCost(depth, currentAmount)
	return swarm.NewBZZ(new(big.Int).Sub(newCost, currentCost)), nil
}

// GetDurationExtensionCost returns the BZZ cost of extending the batch's
// TTL by `duration`.
func (c *Client) GetDurationExtensionCost(ctx context.Context, batchID swarm.BatchID, duration swarm.Duration, opts *StorageOptions) (swarm.BZZ, error) {
	batch, err := c.Postage.GetPostageBatch(ctx, batchID)
	if err != nil {
		return swarm.BZZ{}, err
	}
	price, err := c.chainPrice(ctx)
	if err != nil {
		return swarm.BZZ{}, err
	}
	amount := postage.GetAmountForDuration(duration, price, opts.network().BlockTimeSeconds())
	return swarm.NewBZZ(postage.GetStampCost(int(batch.Depth), amount)), nil
}

// CalculateTopUpForBzz translates a BZZ budget into the amount-per-chunk
// to pass to TopUpBatch and the expected TTL extension that results.
// Mirrors bee-js Bee.calculateTopUpForBzz.
func (c *Client) CalculateTopUpForBzz(ctx context.Context, depth uint8, bzz swarm.BZZ, opts *StorageOptions) (*big.Int, swarm.Duration, error) {
	price, err := c.chainPrice(ctx)
	if err != nil {
		return nil, swarm.ZeroDuration, err
	}
	blockTime := opts.network().BlockTimeSeconds()
	// amount = bzz_PLUR / 2^depth
	denom := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(depth)), nil)
	amount := new(big.Int).Quo(bzz.ToPLURBigInt(), denom)
	dur := postage.GetStampDuration(amount, price, blockTime)
	return amount, dur, nil
}

func labelFromOptions(opts *api.PostageBatchOptions) string {
	if opts == nil {
		return ""
	}
	return opts.Label
}
