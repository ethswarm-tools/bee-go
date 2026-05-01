// Package debug exposes the operator / observability endpoints of a
// Bee node: health and readiness, version + API-version checks,
// addresses / topology / peers, node info and status (full + per-peer
// + per-neighborhood), chain / reserve / redistribution state, balances
// and accounting, settlements, chequebook (balance + cheques + cashout),
// stake (deposit / withdraw / migrate), pending transactions, welcome
// message, and runtime loggers.
//
// Get a [*Service] handle from [github.com/ethswarm-tools/bee-go.Client.Debug].
//
// Mirrors bee-js's Bee.getHealth / getNodeInfo / getStatus / getTopology /
// getPeers / getChainState / getReserveState / getStake / getAllBalances /
// getAllSettlements / getWalletBalance / chequebook + cashout /
// pending-transactions / loggers fan-out, plus a few Bee-only endpoints
// (/accounting, /status/peers, /status/neighborhoods, /connect/{multiaddr})
// that bee-js does not expose today.
package debug
