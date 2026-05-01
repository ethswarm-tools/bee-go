// Package pss implements Swarm Postal Service, the trustless
// publish/subscribe layer that piggy-backs on the chunk-routing topology
// of a Bee swarm.
//
// Get a [*Service] handle from
// [github.com/ethswarm-tools/bee-go.Client.PSS]:
//
//   - PssSend(ctx, batchID, topic, target, data, recipient) — encrypts
//     to recipient (if non-nil) and uploads to a chunk address whose
//     proximity to target controls neighborhood delivery. Bee 2.7+
//     requires a postage batch ID for PSS uploads.
//   - PssSubscribe(ctx, topic) — opens a websocket and returns a
//     [Subscription] with channel-shaped Messages and Errors plus a
//     Cancel() to close it.
//   - PssReceive(ctx, topic, timeout) — convenience: subscribe, return
//     the first message or a timeout error, then close.
//
// Mirrors bee-js's Bee.pssSend / pssSubscribe / pssReceive.
package pss
