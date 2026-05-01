// Package swarm holds the foundational types every other bee-go
// sub-package depends on: typed length-validated byte arrays, token and
// time math, content-addressed chunk primitives, and the typed error
// hierarchy returned by every endpoint.
//
// # Typed bytes
//
// Bee identifiers have fixed lengths and meaningful semantics; bee-go
// represents each one as its own type so the compiler can prevent
// passing the wrong identifier to the wrong endpoint:
//
//	Reference   — 32 or 64 bytes (encrypted refs include the key)
//	BatchID     — 32-byte postage batch identifier
//	EthAddress  — 20-byte Ethereum address
//	PublicKey   — 33-byte compressed secp256k1 public key
//	Signature   — 65-byte (r || s || v) signature
//	Identifier  — 32-byte SOC / feed identifier
//	Topic       — 32-byte hashed PSS / GSOC topic
//	Span        — 8-byte length prefix for chunk payloads
//
// # Token + duration math
//
//   - [BZZ] / [DAI] — fixed-point token wrappers with full arithmetic,
//     comparison, and decimal rendering. Internal representation is
//     PLUR (BZZ base unit) or wei.
//   - [Duration] — nanosecond-precision duration with parser for
//     "1d 4h 5m 30s"-style strings.
//   - [Size]     — byte count with [SizeFromGigabytes] / [SizeFromTerabytes]
//     helpers; used by storage-cost math.
//
// # Chunk + SOC primitives
//
// [MakeContentAddressedChunk] (CAC), [MakeSingleOwnerChunk] (SOC),
// [CalculateSingleOwnerChunkAddress], [GSOCMine] for mining a signer
// whose SOC address lands inside a target neighborhood, and
// [FileChunker] for streaming BMT chunking of arbitrary readers.
//
// # Errors
//
// Every endpoint in bee-go returns nil or one of:
//
//   - *[BeeError]          — base type, network / parse / unexpected
//   - *[BeeArgumentError]  — caller-supplied value is invalid
//   - *[BeeResponseError]  — Bee returned a non-2xx status
//
// Inspect with errors.As, or use the [IsBeeResponseError] helper.
// [CheckResponse] is the one-line helper that bee-go uses internally to
// translate a *http.Response into a typed error and is exported for
// callers who construct their own requests.
package swarm
