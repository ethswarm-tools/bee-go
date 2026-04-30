# bee-go

> A Go client library for connecting to Swarm Bee nodes.

**bee-go** provides a type-safe interface for interacting with the Bee API. It targets functional parity with [bee-js](https://github.com/ethersphere/bee-js) while keeping a Go shape: sub-packages per domain, `context.Context` first arg, errors as values, typed-bytes wrappers for length-validated identifiers.

## Installation

```bash
go get github.com/ethersphere/bee-go
```

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	bee "github.com/ethersphere/bee-go"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

func main() {
	c, err := bee.NewClient("http://localhost:1633")
	if err != nil {
		log.Fatal(err)
	}

	ok, err := c.Debug.Health(context.Background())
	if err != nil || !ok {
		log.Fatal(err)
	}

	// Buy storage for 1 GB / 30 days using current chain pricing.
	size, _ := swarm.SizeFromGigabytes(1)
	batchID, err := c.BuyStorage(context.Background(), size, swarm.DurationFromDays(30), nil)
	if err != nil {
		log.Fatal(err)
	}

	res, err := c.File.UploadData(context.Background(), batchID, strings.NewReader("Hello Swarm!"), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Uploaded reference: %s\n", res.Reference.Hex())
}
```

## Package layout

bee-go is a sub-service client: the top-level `Client` exposes one
sub-service per Bee API domain. This is a deliberate Go idiom — it keeps
each domain's surface focused, allows compiler-checked imports of just
what callers need, and avoids a single 100-method God object. (bee-js
uses one flat `Bee` class because TypeScript has no equivalent of the
import-pruning Go gives us for free.)

| Package | Purpose |
|---|---|
| `pkg/swarm` | Typed bytes (`Reference`, `BatchID`, `EthAddress`, `PublicKey`, `Signature`, …), token math (`BZZ`, `DAI`), `Duration`, `Size`, BMT chunk ops, SOC creation, GSOC mining, typed errors (`BeeError`, `BeeArgumentError`, `BeeResponseError`), `CheckResponse`. |
| `pkg/api` | Core HTTP client + shared options/headers. `UploadOptions`, `RedundantUploadOptions`, `FileUploadOptions`, `CollectionUploadOptions`, `DownloadOptions`, `PostageBatchOptions`. Pin/Tag/Stewardship/Grantee/Envelope endpoints. |
| `pkg/file` | Data, file, chunk, SOC, feed and collection uploads/downloads. `FeedReader`/`FeedWriter`, `MakeFeedIdentifier`, `FeedUpdateChunkReference`, `IsFeedRetrievable`, `AreAllSequentialFeedsUpdateRetrievable`, in-memory `UploadCollectionEntries`. |
| `pkg/postage` | Postage batch CRUD + stamp math (`GetStampCost`, `GetStampDuration`, `GetAmountForDuration`, `GetDepthForSize`, `GetStampEffectiveBytes`). `Stamper` for offline stamp generation. |
| `pkg/debug` | Node info, peers, topology, balances, settlements, chequebook, stake, transactions, redistribution. |
| `pkg/pss` | PSS send/subscribe/receive over WebSockets. |
| `pkg/gsoc` | Generic Single-Owner Chunk send/subscribe (built on `pkg/file` SOC upload). |
| `pkg/manifest` | Mantaray trie + v0.2 wire format (single-chunk). |

Top-level `Client` (in `client.go`) wires every sub-service to the same
HTTP client and base URL. High-level helpers that span multiple
sub-services (e.g. `BuyStorage`, `ExtendStorage`, `GetStorageCost`)
live on `Client` itself in `storage.go`.

## bee-js → bee-go cheat sheet

| bee-js method | bee-go call |
|---|---|
| `bee.uploadData(stamp, data, opts)` | `client.File.UploadData(ctx, batchID, data, opts)` |
| `bee.uploadFile(stamp, data, name, opts)` | `client.File.UploadFile(ctx, batchID, data, name, contentType, opts)` |
| `bee.uploadFiles(stamp, files, opts)` | `client.File.UploadCollectionEntries(ctx, batchID, entries, opts)` |
| `bee.uploadFilesFromDirectory(stamp, dir, opts)` | `client.File.UploadCollection(ctx, batchID, dir, opts)` |
| `bee.uploadChunk(stamp, data, opts)` | `client.File.UploadChunk(ctx, batchID, data, opts)` |
| `bee.downloadData(ref, opts)` | `client.File.DownloadData(ctx, ref, opts)` |
| `bee.downloadFile(ref, path, opts)` | `client.File.DownloadFile(ctx, ref, opts)` |
| `bee.downloadChunk(ref, opts)` | `client.File.DownloadChunk(ctx, ref, opts)` |
| `bee.uploadSOC(stamp, owner, id, sig, data, opts)` | `client.File.UploadSOC(ctx, batchID, owner, id, sig, data, opts)` |
| `bee.createFeedManifest(stamp, topic, owner)` | `client.File.CreateFeedManifest(ctx, batchID, owner, topic)` |
| `bee.makeFeedReader(topic, owner)` | `client.File.MakeFeedReader(owner, topic)` |
| `bee.makeFeedWriter(topic, signer)` | `client.File.MakeFeedWriter(signer, topic)` |
| `bee.fetchLatestFeedUpdate(owner, topic)` | `client.File.FetchLatestFeedUpdate(ctx, owner, topic)` |
| `bee.isReferenceRetrievable(ref)` | `client.API.IsRetrievable(ctx, ref)` |
| `bee.isFeedRetrievable(owner, topic, idx)` | `client.File.IsFeedRetrievable(ctx, owner, topic, &idx, opts)` |
| `bee.createPostageBatch(amount, depth, opts)` | `client.Postage.CreatePostageBatch(ctx, amount, depth, label)` |
| `bee.topUpBatch(id, amount)` | `client.Postage.TopUpBatch(ctx, batchID, amount)` |
| `bee.diluteBatch(id, depth)` | `client.Postage.DiluteBatch(ctx, batchID, depth)` |
| `bee.getPostageBatch(id)` / `getAllPostageBatch()` | `client.Postage.GetPostageBatch(ctx, batchID)` / `GetPostageBatches(ctx)` |
| `bee.buyStorage(size, duration, opts)` | `client.BuyStorage(ctx, size, duration, opts)` |
| `bee.getStorageCost(size, duration)` | `client.GetStorageCost(ctx, size, duration, opts)` |
| `bee.extendStorage(id, size, duration)` | `client.ExtendStorage(ctx, batchID, size, duration, opts)` |
| `bee.extendStorageSize(id, size)` | `client.ExtendStorageSize(ctx, batchID, size, opts)` |
| `bee.extendStorageDuration(id, duration)` | `client.ExtendStorageDuration(ctx, batchID, duration, opts)` |
| `bee.getExtensionCost(id, size, duration)` | `client.GetExtensionCost(ctx, batchID, size, duration, opts)` |
| `bee.getSizeExtensionCost(id, size)` | `client.GetSizeExtensionCost(ctx, batchID, size, opts)` |
| `bee.getDurationExtensionCost(id, duration)` | `client.GetDurationExtensionCost(ctx, batchID, duration, opts)` |
| `bee.calculateTopUpForBzz(depth, bzz)` | `client.CalculateTopUpForBzz(ctx, depth, bzz, opts)` |
| `bee.pin(ref)` / `unpin(ref)` | `client.API.Pin(ctx, ref)` / `Unpin(ctx, ref)` |
| `bee.getPin(ref)` / `getAllPins()` | `client.API.GetPin(ctx, ref)` / `ListPins(ctx)` |
| `bee.createTag()` / `getTag(uid)` | `client.API.CreateTag(ctx)` / `GetTag(ctx, uid)` |
| `bee.reuploadPinnedData(stamp, ref)` | `client.API.Reupload(ctx, ref, batchID)` |
| `bee.pssSend(stamp, topic, target, data, recipient)` | `client.PSS.PssSend(ctx, topic, target, data, recipient)` |
| `bee.pssSubscribe(topic, handler)` | `client.PSS.PssSubscribe(ctx, topic)` (returns `Subscription{Messages, Errors}`) |
| `bee.pssReceive(topic, timeoutMs)` | `client.PSS.PssReceive(ctx, topic, timeout)` |
| `bee.gsocMine(overlay, id, prox)` | `swarm.GSOCMine(target, identifier, proximity)` |
| `bee.gsocSend(stamp, signer, id, data)` | `client.GSOC.Send(ctx, batchID, signer, id, data, opts)` |
| `bee.gsocSubscribe(addr, id, handler)` | `client.GSOC.Subscribe(ctx, owner, id)` |
| `bee.getNodeInfo()` | `client.Debug.NodeInfo(ctx)` |
| `bee.getStatus()` | `client.Debug.Status(ctx)` |
| `bee.getNodeAddresses()` | `client.Debug.Addresses(ctx)` |
| `bee.getTopology()` | `client.Debug.Topology(ctx)` |
| `bee.getPeers()` | `client.Debug.Peers(ctx)` |
| `bee.getChainState()` | `client.Debug.ChainState(ctx)` |
| `bee.getReserveState()` | `client.Debug.ReserveState(ctx)` |
| `bee.getRedistributionState()` | `client.Debug.RedistributionState(ctx)` |
| `bee.getStake()` / `stake(amount)` | `client.Debug.GetStake(ctx)` / `Stake(ctx, amount)` |
| `bee.getWithdrawableStake()` / `withdrawSurplusStake()` | `client.Debug.GetWithdrawableStake(ctx)` / `WithdrawSurplusStake(ctx)` |
| `bee.migrateStake()` | `client.Debug.MigrateStake(ctx)` |
| `bee.getBalances()` / `getBalance(peer)` | `client.Debug.GetBalances(ctx)` / `GetPeerBalance(ctx, peer)` |
| `bee.getPastDueConsumptionBalances()` | `client.Debug.GetPastDueConsumptionBalances(ctx)` |
| `bee.getPastDueConsumptionPeerBalance(peer)` | `client.Debug.GetPastDueConsumptionPeerBalance(ctx, peer)` |
| `bee.getSettlements(peer)` | `client.Debug.PeerSettlement(ctx, peer)` |
| `bee.getAllSettlements()` | `client.Debug.Settlements(ctx)` |
| `bee.getChequebookAddress()` / `getChequebookBalance()` | (chequebook fields on `client.Debug.GetWallet`) / `client.Debug.GetChequebookBalance(ctx)` |
| `bee.depositTokens(amount)` / `withdrawTokens(amount)` | `client.Debug.DepositTokens(ctx, amount)` / `WithdrawTokens(ctx, amount)` |
| `bee.getLastCheques()` | `client.Debug.LastCheques(ctx)` |
| `bee.getAllPendingTransactions()` | `client.Debug.GetAllPendingTransactions(ctx)` |
| `bee.getPendingTransaction(hash)` | `client.Debug.GetPendingTransaction(ctx, hash)` |
| `bee.rebroadcastPendingTransaction(hash)` | `client.Debug.RebroadcastPendingTransaction(ctx, hash)` |
| `bee.cancelPendingTransaction(hash, gasPrice)` | `client.Debug.CancelPendingTransaction(ctx, hash, gasPrice)` |

Construct a dev-mode client with `bee.NewDevClient(url)` — it returns
the same surface, but most chain/payment endpoints will return a
`*BeeResponseError` 404 against a `bee dev` node.

## Errors

Every endpoint returns either nil or a `*swarm.BeeError`,
`*swarm.BeeArgumentError`, or `*swarm.BeeResponseError`. Inspect with
`errors.As`:

```go
if rerr, ok := swarm.IsBeeResponseError(err); ok {
    fmt.Printf("Bee returned %d %s for %s %s\n",
        rerr.Status, rerr.StatusText, rerr.Method, rerr.URL)
    fmt.Printf("Body: %s\n", rerr.ResponseBody)
}
```

`swarm.CheckResponse(resp)` is a one-line helper used internally and
available to callers who construct their own requests.

## Examples

`examples/` contains short runnable programs:

- `examples/basic-usage` — health check + node info
- `examples/buy-batch` — purchase a postage batch
- `examples/upload-picture` — upload a file
- `examples/download-picture` — download a file
- `examples/status` — read chain/reserve/redistribution state

## Contribute

Contributions are welcome — please open an issue first for anything
substantial. Run `go test ./...` before submitting; the test suite is
self-contained (no live Bee node required).

## License

MIT
