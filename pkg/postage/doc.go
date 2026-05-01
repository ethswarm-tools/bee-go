// Package postage covers the Swarm postage-batch lifecycle plus the
// pure stamp math used to translate (size, duration) into (depth,
// amount) and back.
//
// Get a [*Service] handle from
// [github.com/ethswarm-tools/bee-go.Client.Postage] for batch CRUD:
//
//   - CreatePostageBatch / CreatePostageBatchWithOptions
//   - TopUpBatch, DiluteBatch
//   - GetPostageBatch (one), GetPostageBatches (owned, hits /stamps)
//   - GetGlobalPostageBatches (chain-wide, hits /batches; alias
//     GetAllGlobalPostageBatch is deprecated)
//   - GetPostageBatchBuckets
//
// Stamp math is exposed as free functions because it has no I/O:
// [GetStampCost], [GetStampDuration], [GetAmountForDuration],
// [GetDepthForSize], [GetStampEffectiveBytes].
//
// [Stamper] is an offline postage stamper for callers that need to sign
// stamps client-side (e.g. when uploading via /chunks or /soc with a
// pre-issued envelope). [MarshalStamp] /
// [ConvertEnvelopeToMarshaledStamp] produce the 113-byte on-wire stamp
// layout (batchID || index || timestamp || signature).
//
// Mirrors bee-js's createPostageBatch / topUpBatch / diluteBatch /
// getPostageBatch / getGlobalPostageBatches / Stamper / marshalStamp /
// convertEnvelopeToMarshaledStamp surface.
//
// # Batch-usability delay
//
// A postage batch is not usable for uploads immediately after
// CreatePostageBatch returns. Bee waits for N confirmations of the
// purchase transaction (configurable on the node, typically 8) before
// flipping the batch's usable bit. On Gnosis (5-second blocks) this
// is ~40 seconds; on Sepolia (12-second blocks) it can be 2-3 minutes
// and occasionally longer.
//
// Use the returned [PostageBatch.Usable] field — poll
// [Service.GetPostageBatch] in a loop until Usable is true, or supply
// the batch via an environment variable so tests can reuse a known-good
// batch. Uploading against a not-yet-usable batch returns
// HTTP 422 "stamp not usable".
//
// # Dilute is one-way
//
// [Service.DiluteBatch] only allows depth to grow. Once a batch's depth
// is increased its previously-issued stamps remain valid; there is no
// way to shrink depth and reclaim funds.
package postage
