# Changelog

All notable changes to this project are documented here. The format is
based on [Keep a Changelog](https://keepachangelog.com/) and this
project adheres to [Semantic Versioning](https://semver.org/) once
`v1.0.0` is tagged. Pre-1.0 releases may break the API on a minor
version bump.

## [Unreleased]

### Added

- GitHub Actions CI workflow (`.github/workflows/ci.yml`) running
  `go vet`, `go test -race`, and `golangci-lint` on every PR and push
  to `main`.
- `.golangci.yml` baseline: errcheck, govet, ineffassign, staticcheck,
  unused, gosec, gocritic, revive, misspell, gofmt, goimports.
- `RELEASE.md` documenting the tag flow, pre-release checklist, and
  CHANGELOG conventions.

### Changed

- **BREAKING:** `Tag.Uid` → `Tag.UID`, `UploadResult.TagUid` →
  `TagUID`, `FileHeaders.TagUid` → `TagUID`. JSON wire tags are
  unchanged (`"uid"`).
- **BREAKING:** `debug.SupportedApiVersion` → `SupportedAPIVersion`,
  `debug.BeeVersions.SupportedBeeApiVersion` → `SupportedBeeAPIVersion`,
  `BeeVersions.BeeApiVersion` → `BeeAPIVersion`,
  `(*debug.Service).IsSupportedApiVersion` → `IsSupportedAPIVersion`.
- `BeeResponseError` no longer prints the HTTP status code twice (was
  `… 422 422 Unprocessable Entity`, now `… 422 Unprocessable Entity`).
- `pkg/manifest`: `MantarayNode.CalculateSelfAddress` and
  `SaveRecursively` now stream the marshaled node through
  `swarm.FileChunker` so nodes whose marshal exceeds `ChunkSize` are
  chunked transparently. Previously these returned an error.

## [0.1.0] — 2026-04-30

First public preview. Establishes feature parity with bee-js, plus a
few Bee-only operator endpoints. API surface is locked in spirit but
may still receive small breaking renames before `v1.0.0`.

### Added — bee-js parity surface

- **Top-level `Client`** with sub-services: `API`, `Debug`, `File`,
  `Postage`, `Swarm`, `PSS`, `GSOC`. `NewClient(url, opts...)`,
  `NewDevClient(url, opts...)`.
- **`pkg/swarm`** — typed bytes (`Reference`, `BatchID`, `EthAddress`,
  `PublicKey`, `Signature`, `Identifier`, `Topic`, …), token math
  (`BZZ`, `DAI` with Plus/Minus/Divide/Cmp/exchange), `Duration`,
  `Size`, BMT chunk addressing, SOC creation/unmarshaling/recovery,
  GSOC mining + proximity, content-addressed chunk constructors
  (`MakeContentAddressedChunk`, `MakeSingleOwnerChunk`,
  `CalculateSingleOwnerChunkAddress`), streaming `FileChunker`, typed
  errors (`BeeError`, `BeeArgumentError`, `BeeResponseError`,
  `CheckResponse`).
- **`pkg/api`** — `UploadOptions` + variants (Redundant, File,
  Collection); pin / tag / stewardship / grantee / envelope endpoints;
  `IsRetrievable`, `Reupload`. Tag CRUD + `RetrieveTag` alias.
- **`pkg/file`** — Data, file, chunk, SOC, feed and collection
  uploads/downloads. `FeedReader`/`FeedWriter`, `MakeFeedIdentifier`,
  `FeedUpdateChunkReference`, `IsFeedRetrievable`,
  `AreAllSequentialFeedsUpdateRetrievable`. In-memory
  `UploadCollectionEntries` (tar-stream). `ProbeData(ref)`. SOC
  reader/writer (`MakeSOCReader` / `MakeSOCWriter`). `HashDirectory`
  / `HashCollectionEntries` (offline content addressing).
  `StreamDirectory` / `StreamCollectionEntries` (chunk-by-chunk
  upload with progress callback).
- **`pkg/postage`** — Postage batch CRUD; `CreatePostageBatch`,
  `TopUpBatch`, `DiluteBatch`, `GetPostageBatch`, `GetPostageBatches`
  (owned, hits `/stamps`), `GetGlobalPostageBatches` (chain-wide,
  hits `/batches`, with `GetAllGlobalPostageBatch` deprecated alias),
  `GetPostageBatchBuckets`. Stamp math (`GetStampCost`,
  `GetStampDuration`, `GetAmountForDuration`, `GetDepthForSize`,
  `GetStampEffectiveBytes`). `Stamper` for offline stamp generation.
  `MarshalStamp` / `ConvertEnvelopeToMarshaledStamp` for the wire
  format.
- **`pkg/debug`** — Health + structured `GetHealth`, `GetVersions`,
  `IsSupportedAPIVersion`, `IsSupportedExactVersion`, `IsConnected` /
  `CheckConnection`, `IsGateway`, `Readiness`. Node info, status,
  addresses, topology, peers, chain state, reserve state,
  redistribution state. Wallet, chequebook (balance + cheques +
  cashout: `GetLastChequesForPeer`, `GetLastCashoutAction`,
  `CashoutLastCheque`), settlements, accounting, balances. Stake +
  `WithdrawSurplusStake`, `MigrateStake`, `DepositStake` alias.
  Pending transactions (full lifecycle). Bee-only operator
  endpoints: `/accounting`, `/status/peers`, `/status/neighborhoods`,
  `/connect/{multi-address}`, `/welcome-message`, `/loggers` trio.
  bee-js name aliases (`WithdrawBZZToExternalWallet`,
  `WithdrawDAIToExternalWallet`, `DepositBZZToChequebook`,
  `WithdrawBZZFromChequebook`).
- **`pkg/pss`** — PSS send / subscribe / receive over WebSockets,
  channel-shaped `Subscription{Topic, Messages, Errors}`.
- **`pkg/gsoc`** — GSOC send + subscribe; `SOCAddress` (offline
  reference computation).
- **`pkg/manifest`** — Mantaray trie with the v0.2 wire format;
  `New`, `AddFork`, `Find`, `FindClosest`, `RemoveFork`, `Marshal`,
  `Unmarshal`, `CalculateSelfAddress`, `Collect`, `CollectAndMap`.
  `SaveRecursively(ctx, uploader, batchID)` for chunk-by-chunk
  manifest publication.
- **Top-level helpers** that span multiple sub-services:
  `BuyStorage`, `GetStorageCost`, `ExtendStorage`, `ExtendStorageSize`,
  `ExtendStorageDuration`, `GetExtensionCost`, `GetSizeExtensionCost`,
  `GetDurationExtensionCost`, `CalculateTopUpForBzz`. `Network`
  (Gnosis = 5s blocks, Mainnet = 15s).

### Added — verified live

- `examples/integration-check` — sequential smoke test against a
  real Bee node. Set `BEE_URL` (default `http://localhost:1633`) and
  optionally `BEE_BATCH_ID` to reuse an existing usable batch
  instead of buying a new one. Last full run: 53 / 54 checks pass
  against Bee 2.7.1 on Sepolia (the single failure is a
  server-side `/balances` 500 unrelated to the client).

### Fixed — bugs surfaced only against a live Bee

- `ChainStateResponse.CurrentPrice` was `uint64` with `json` tag
  but Bee returns it (and `totalAmount`) as bigint-encoded
  strings. Custom `UnmarshalJSON` parses them; `TotalAmount` field
  added.
- `swarm.Sign` (used by `CreateSOC`) hashed with raw keccak256 and
  left signature V at {0,1}. Bee verifies SOC signatures against
  the Ethereum signed-message digest with V∈{27,28}. Both the
  signer and the matching `UnmarshalSingleOwnerChunk` recovery
  path now use the eth-signed-message digest and the correct V
  encoding.
