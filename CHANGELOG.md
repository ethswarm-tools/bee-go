# Changelog

All notable changes to this project are documented here. The format is
based on [Keep a Changelog](https://keepachangelog.com/) and this
project adheres to [Semantic Versioning](https://semver.org/) once
`v1.0.0` is tagged. Pre-1.0 releases may break the API on a minor
version bump.

## [Unreleased]

### Added

- **godoc landing page.** Root `doc.go` adds a package-level overview
  with a quickstart snippet, a sub-package map, dev-mode notes, and
  pointers to the error model and `examples/`. Previously the
  pkg.go.dev landing page rendered only the type list with no
  prose — now it opens with a complete onboarding read.
- **Per-subpackage doc.go.** New `doc.go` in each of `pkg/api`,
  `pkg/debug`, `pkg/file`, `pkg/postage`, `pkg/pss`, `pkg/swarm`
  giving each sub-package an overview, headline-piece list, and
  bee-js mirrors statement. (`pkg/gsoc` and `pkg/manifest` already
  had package-level docs and are unchanged.)
- **`example_test.go`.** Pkg.go.dev-rendered `Example*` functions for
  `NewClient` (health check), `Client.UploadData`, `Client.DownloadData`,
  `Client.BuyStorage`, and the typed-error inspection pattern. These
  show up inline alongside the symbol on pkg.go.dev.
- **Type / option doc upgrades.** Replaced the template-y "X handles Y
  operations" doc on `Client`, `*api.Service`, `*debug.Service`,
  `*file.Service`, `*postage.Service`, `*pss.Service`, `*swarm.Service`
  with one-paragraph descriptions that name the headline endpoints.
  `Client` field comments now describe each sub-service. `NewClient`,
  `WithHTTPClient`, and `ClientOption` got expanded prose covering the
  defaults, when to override, and what the contract is.
- **Operational sections in root `doc.go`.** Added Bee version
  compatibility (pinned to 2.7.1 / API 7.4.1), authentication +
  timeouts + proxies (with `WithHTTPClient` snippet), concurrency,
  cancellation, streaming vs. buffered transfers, errors-and-
  retryability, observability, testing (with `httptest.Server`
  example), common pitfalls (batch usability, dilute one-way,
  encrypted-vs-plain references, feed signer pairing, dev-mode 404s),
  and Go version (1.25+).
- **Postage usability + dilute-one-way notes** in `pkg/postage/doc.go`:
  paragraph on the ~2-3 minute Sepolia delay before a batch flips
  `Usable: true`, and a paragraph on `DiluteBatch` being one-way.
- **File streaming notes** in `pkg/file/doc.go`: streaming-by-default
  download semantics, the `io.Copy` vs. `io.ReadAll` OOM warning, and
  cancellation behavior of `StreamDirectory` /
  `StreamCollectionEntries`.
- **Dev-mode 404 list** in `dev.go`: explicit list of every endpoint
  that returns 404 against `bee dev` (chequebook, settlements, stake,
  pending transactions, chain-state reads, accounting, balances, RC
  hash, and the high-level helpers that internally call them).

## [1.0.2] — 2026-05-01

### Added

- README "Package layout" table now links each `pkg/*` row directly
  to its pkg.go.dev page, plus a one-line "Full API reference"
  pointer above the top-level `Client` description. The root
  pkg.go.dev page only renders the thin top-level `Client` wrapper
  (Client, ClientOption, NewClient, BuyStorage, ExtendStorage,
  GetStorageCost) — the bulk of the surface (95 types, 31 free
  functions, plus all sub-service methods) lives in the sub-package
  pages, which were previously discoverable only via pkg.go.dev's
  Directories tree. Direct links cut the discovery step.

## [1.0.1] — 2026-05-01

### Added

- `LICENSE` file at the repository root with the MIT license text (the
  `README.md` already declared MIT but the file was missing). Required
  by pkg.go.dev's redistributable-license policy — without a license
  file at the module root, full Go-doc rendering is suppressed and
  the page only shows the directory tree. With this file in place,
  `https://pkg.go.dev/github.com/ethswarm-tools/bee-go@v1.0.1` renders
  the full package, type, and function documentation.

### Fixed (CI, on `main` since v1.0.0)

- Bumped `golangci/golangci-lint-action` v6 → v7 for golangci-lint
  v2 support.
- Pinned `golangci-lint` to `v2.11.3` (built with go1.26) so the CI
  matrix's stable Go entry doesn't pull a forward-incompatible
  build.

## [1.0.0] — 2026-04-30

First stable release. SemVer compatibility promise is now in effect for
everything exported from the top-level module and `pkg/...`.

### Added

- GitHub Actions CI workflow (`.github/workflows/ci.yml`) running
  `go vet`, `go test -race`, and `golangci-lint` on every PR and push
  to `main`.
- `.golangci.yml` baseline: errcheck, govet, ineffassign, staticcheck,
  unused, gosec, gocritic, revive, misspell, gofmt, goimports.
- `RELEASE.md` documenting the tag flow, pre-release checklist, and
  CHANGELOG conventions.
- `examples/integration-check` extended with live scenarios for feeds
  (UpdateFeed → FetchLatestFeedUpdate → IsFeedRetrievable), PSS
  (subscribe + send loopback), GSOC (mine + subscribe + send), encryption
  + redundancy upload round-trip, and `TopUpBatch` lifecycle.
- `PostageBatch.Exists bool` (the `exists` field that Bee returns from
  `/stamps/{id}`).

### Changed

- **BREAKING:** Module path renamed `github.com/ethersphere/bee-go` →
  `github.com/ethswarm-tools/bee-go` to match the canonical GitHub
  repository. Update imports accordingly.
- **BREAKING:** `Tag.Uid` → `Tag.UID`, `UploadResult.TagUid` →
  `TagUID`, `FileHeaders.TagUid` → `TagUID`. JSON wire tags are
  unchanged (`"uid"`).
- **BREAKING:** `debug.SupportedApiVersion` → `SupportedAPIVersion`,
  `debug.BeeVersions.SupportedBeeApiVersion` → `SupportedBeeAPIVersion`,
  `BeeVersions.BeeApiVersion` → `BeeAPIVersion`,
  `(*debug.Service).IsSupportedApiVersion` → `IsSupportedAPIVersion`.
- **BREAKING:** `(*pss.Service).PssSend` now requires a `batchID
  swarm.BatchID` parameter. Bee 2.7+ rejects PSS uploads without
  `Swarm-Postage-Batch-Id`; the previous signature could never succeed
  against a live node.
- **BREAKING:** `postage.PostageBatch.Value *big.Int` → `Amount
  *big.Int`. Bee returns the per-chunk amount as `"amount"` on
  `/stamps` and `/stamps/{id}`; the old `"value"` mapping left the
  field nil for every owned-batch read. The chain-wide
  `GlobalPostageBatch.Value` (from `/batches`) is unchanged.
- `BeeResponseError` no longer prints the HTTP status code twice (was
  `… 422 422 Unprocessable Entity`, now `… 422 Unprocessable Entity`).
- `pkg/manifest`: `MantarayNode.CalculateSelfAddress` and
  `SaveRecursively` now stream the marshaled node through
  `swarm.FileChunker` so nodes whose marshal exceeds `ChunkSize` are
  chunked transparently. Previously these returned an error.
- Pinned `SupportedBeeVersionExact` → `2.7.1-61fab37b`,
  `SupportedAPIVersion` → `7.4.1` (the version the live integration
  check now passes against).

### Fixed

- `(*file.Service).UpdateFeedWithIndex` and `(*gsoc.Service).Send`
  uploaded only the SOC payload to `/soc/{owner}/{id}` — Bee then
  computed the CAC over `payload` instead of `span || payload`,
  signature verification failed, and every live call returned 401.
  Both now upload `span || payload` (matching `SOCWriter.Upload`).
- `postage.PostageBatch` `Immutable bool` was tagged `"immutable"` but
  Bee returns `"immutableFlag"`; the field was always false. Tag
  corrected.

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
