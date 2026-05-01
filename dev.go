package bee

// DevClient is a Bee client variant that talks to a Bee node running in
// "dev" mode (`bee dev`). It wraps the regular Client but documents the
// reduced surface — most chain-state, chequebook, settlement, postage
// purchase and stake endpoints are not available on dev nodes and will
// return a *BeeResponseError with status 404 if called.
//
// # Endpoints that work
//
//   - Addresses / Topology / NodeInfo / Status (with dev-shaped, simpler
//     JSON — the existing parsers tolerate the missing fields)
//   - Health, Readiness
//   - File upload/download (/bytes, /bzz, /chunks, /soc, /feeds)
//   - PSS subscribe/send
//   - GSOC subscribe/send
//   - Tags, Pins, Stewardship, Grantees, Envelopes
//   - The /stamps endpoints behave as no-ops in dev mode but do not 404
//
// # Endpoints that return 404
//
//   - All chequebook endpoints — balance, deposit, withdraw, cheques,
//     cashout (Service.GetChequebookBalance, .DepositTokens,
//     .WithdrawTokens, .LastCheques, .GetLastChequesForPeer,
//     .GetLastCashoutAction, .CashoutLastCheque).
//   - All settlement endpoints — .Settlements, .PeerSettlement.
//   - All stake endpoints — .GetStake, .Stake, .DepositStake,
//     .WithdrawSurplusStake, .MigrateStake, .GetWithdrawableStake.
//   - Pending-transaction lifecycle — .GetAllPendingTransactions,
//     .GetPendingTransaction, .RebroadcastPendingTransaction,
//     .CancelPendingTransaction.
//   - Chain-state reads — .ChainState, .ReserveState,
//     .RedistributionState.
//   - The redistribution / RC-hash endpoint .RCHash.
//   - The /accounting and /balances endpoints (per-peer accounting
//     does not exist on a single-node dev instance).
//   - High-level helpers that internally call any of the above —
//     Client.BuyStorage, .GetStorageCost, .ExtendStorage*, etc.
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
