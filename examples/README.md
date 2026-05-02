# bee-go examples

Forty runnable programs that show how to use the bee-go client
against a live Bee node. Each example lives in its own
`examples/<name>/main.go` and runs via `go run ./examples/<name>/ -- <args>`.

The examples are grouped by tier:

- [Setup](#setup) — what you need before running anything.
- [Quickstart & basics](#quickstart--basics) — the ~5 examples to read first.
- [Tier A — feature demos](#tier-a--feature-demos) — one example per
  primitive (manifests, ACT, SOC, GSOC, feeds, stamps, …).
- [Tier B — starter projects](#tier-b--starter-projects) — small CLI
  tools (`swarm-paste`, `swarm-deploy`, `swarm-vault`, …) that
  combine multiple primitives into something you'd actually ship.

Every example prints a usage line on `--help` (or on missing
arguments). Read the file header — each one starts with a doc
comment explaining what it does, the exact CLI shape, and the env
vars it reads.

---

## Setup

All examples accept these environment variables:

| Variable | Purpose | Default |
|---|---|---|
| `BEE_URL` | Bee node base URL | `http://localhost:1633` |
| `BEE_BATCH_ID` | Hex postage batch ID for uploads | required for any upload |
| `BEE_SIGNER_HEX` | 32-byte hex private key | required for feeds, SOC, GSOC, ACT |

Generate a signer once:

```sh
openssl rand -hex 32 > .signer
export BEE_SIGNER_HEX=$(cat .signer)
```

Buy a batch once (or reuse an existing one — see
[`buy-batch`](buy-batch/main.go)):

```sh
go run ./examples/buy-batch
export BEE_BATCH_ID=<the hex printed above>
```

> On Sepolia the first usability of a fresh batch takes several
> minutes. Reuse `BEE_BATCH_ID` whenever possible.

---

## Quickstart & basics

Read these in order if you're new to bee-go.

| Example | What it shows |
|---|---|
| [`basic-usage`](basic-usage/main.go) | Health and node-info round-trip. |
| [`status`](status/main.go) | Pretty-print `Status`, `Health`, `NodeInfo`. |
| [`buy-batch`](buy-batch/main.go) | Buy a postage batch and wait until it's usable. |
| [`upload-picture`](upload-picture/main.go) / [`download-picture`](download-picture/main.go) | The "hello world" of file uploads. |
| [`integration-check`](integration-check/main.go) | Sanity-check a Bee node before running other examples. |

---

## Tier A — feature demos

One example per primitive. Most run against any Bee node (no Sepolia
traffic required) given a usable batch.

### Pinning & retrievability

| Example | Pattern |
|---|---|
| [`pinning-workflow`](pinning-workflow/main.go) | `Pin` → `ListPins` → `IsRetrievable` → `Reupload` → `Unpin` → re-pin |
| [`tag-upload-progress`](tag-upload-progress/main.go) | Tag-tracked deferred upload + progress polling |

### Manifests & encrypted folders

| Example | Pattern |
|---|---|
| [`manifest-add-file`](manifest-add-file/main.go) | Build a manifest offline, upload, verify path serves |
| [`manifest-move-file`](manifest-move-file/main.go) | `AddFork` / `RemoveFork` to rename a path |
| [`encrypted-upload`](encrypted-upload/main.go) | `Encrypt: true` upload, 64-byte ref round-trip |
| [`encrypted-folder-walk`](encrypted-folder-walk/main.go) | Walk an encrypted manifest by downloading the root chunk + offline `Unmarshal` |

### Feeds

| Example | Pattern |
|---|---|
| [`feed-update`](feed-update/main.go) | `UpdateFeed` → `FetchLatestFeedUpdate` |
| [`feed-manifest`](feed-manifest/main.go) | `CreateFeedManifest` → stable `/bzz/<feedRef>/` URL across updates |
| [`feed-history`](feed-history/main.go) | Walk `0..N` indexes via `MakeFeedIdentifier` + SOC reader |

### Single-Owner Chunks & GSOC

| Example | Pattern |
|---|---|
| [`soc-write-read`](soc-write-read/main.go) | `MakeSOCWriter` → upload at distinct identifiers → verify via reader |
| [`gsoc-mined-pubsub`](gsoc-mined-pubsub/main.go) | `GSOCMine` for the local overlay → `Subscribe` + `Send` → receive |

### Access Control Trie (ACT)

| Example | Pattern |
|---|---|
| [`act-share`](act-share/main.go) | Upload with `Act: true` → `CreateGrantees` → `PatchGrantees(add, revoke)` → download as publisher |

### PSS

| Example | Pattern |
|---|---|
| [`pss-send-receive`](pss-send-receive/main.go) | `PssSubscribe` + `PssSend` over a topic prefix |

### Postage stamps

| Example | Pattern |
|---|---|
| [`list-batches`](list-batches/main.go) | Tabular `GetPostageBatches` with TTL/utilization/usable/immutable flags |
| [`stamp-utilization`](stamp-utilization/main.go) | Per-bucket fill via `GetPostageBatchBuckets` |
| [`stamp-cost`](stamp-cost/main.go) | Offline cost projection (`GetStampCost`) |
| [`stamp-cost-live`](stamp-cost-live/main.go) | Live `ChainState` price → full preview |
| [`chain-state`](chain-state/main.go) | Block, chain tip, current price, total amount |
| [`stamper-client-side`](stamper-client-side/main.go) | Client-side `Stamper` (in-memory bucket counters) |
| [`redundant-upload`](redundant-upload/main.go) | Upload with each `RedundancyLevel`, show overhead |

### Offline conversion / utilities

| Example | Pattern |
|---|---|
| [`ref-to-cid`](ref-to-cid/main.go) | `ConvertReferenceToCID` (manifest + feed codecs) |
| [`key-gen`](key-gen/main.go) | Generate secp256k1 key, derive Ethereum address |

> Note: bee-go does not yet expose a `ResourceLocator` /
> `FromENS` helper, so there is no `ens-locator` example here. See
> the bee-rs companion (`ens-locator.rs`) for that flow.

### Misc

| Example | Pattern |
|---|---|
| [`upload-directory`](upload-directory/main.go) | `UploadCollection` from a local directory |

---

## Tier B — starter projects

These compose multiple primitives into mini CLI apps. Each one
maintains a small JSON state file in the working directory; treat
them as scaffolds you'd fork and extend.

| Example | Pitch | Composes |
|---|---|---|
| [`swarm-paste`](swarm-paste/main.go) | Pastebin: stdin → upload → `/bzz/<ref>/` URL. | `UploadFile` |
| [`swarm-deploy`](swarm-deploy/main.go) | `git push`-style site deploy with feed manifest + history rollback. | `UploadCollection` + feed manifest |
| [`swarm-blog`](swarm-blog/main.go) | Markdown blog with one stable URL, regenerated HTML on each `publish`. | collections + feed manifest |
| [`swarm-fs`](swarm-fs/main.go) | Filesystem-style staging tree (`add` / `mv` / `rm`) → `publish` builds a manifest. | local JSON + collection upload |
| [`swarm-chat`](swarm-chat/main.go) | Terminal chat over PSS with username envelopes (run two instances). | PSS subscribe + send |
| [`swarm-vault`](swarm-vault/main.go) | Personal encrypted dropbox with stable URL. | encrypted upload + index manifest + feed |
| [`swarm-share`](swarm-share/main.go) | Revocable file sharing (`share` / `revoke` / `grantees`). | ACT upload + grantee CRUD |
| [`swarm-pinner`](swarm-pinner/main.go) | Daemon: watch dir, upload+pin new files, retrievability poll. | pin + `IsRetrievable` |
| [`swarm-feed-rss`](swarm-feed-rss/main.go) | Read-only aggregator over N feeds (no signer needed). | feeds |
| [`swarm-relay`](swarm-relay/main.go) | Single-batch upload gateway with persisted bucket bookkeeping. | client-side `Stamper` + chunker |
| [`swarm-keyring`](swarm-keyring/main.go) | Passphrase-encrypted secp256k1 key store (scrypt + AES-GCM). | offline crypto |
| [`swarm-cost-monitor`](swarm-cost-monitor/main.go) | Operator dashboard: batch TTLs, live price, projected refill cost. | `GetPostageBatches` + `ChainState` |

---

## Caveats and gotchas

- **PSS doesn't loop back.** A node won't receive its own PSS
  messages. To exercise [`pss-send-receive`](pss-send-receive/main.go)
  or [`swarm-chat`](swarm-chat/main.go), run two instances against
  different Bee URLs (or different batches).
- **GSOC mining lands chunks in the local neighbourhood.** If your
  Bee node has no peers (single-node dev), GSOC pubsub still works
  because every chunk is "local" in a 1-node network.
- **ACT downloads need the publisher identity.** Bee resolves ACT
  permissions using the *node's* key; on a single-node setup, you
  can download as the publisher but cannot exercise the
  grantee-downloads-as-different-identity flow without a second
  node.
- **Sepolia batch usability is slow.** Reuse `BEE_BATCH_ID` whenever
  possible. First-time usability of a fresh batch can take several
  minutes.
- **Mutable feed-manifest URLs** stay the same across updates;
  per-update upload references change. Use the manifest URL for
  sharing, the per-update reference for verification.
- **bee-go's `Stamper` is in-memory only.** There is no public state
  hydrator; see [`swarm-relay`](swarm-relay/main.go) for the workaround
  (replay counters via dummy stamps). bee-rs's `Stamper` exposes
  `from_state` directly.

---

## Adding a new example

1. Create `examples/<name>/main.go` with a top-of-file doc comment
   covering: purpose, CLI shape (`go run ./examples/<name>/ -- …`),
   and required env vars.
2. Use `package main` and a `run() error` helper, matching the
   existing examples.
3. Verify with `go build ./examples/<name>/`.

The header doc comment is the example's documentation — keep it
honest and runnable.
