// Package bee is a Go client library for connecting to Swarm Bee nodes.
//
// It targets functional parity with [bee-js] (the canonical TypeScript
// client) while keeping a Go shape: sub-packages per Bee API domain,
// context.Context as first arg, errors as values, typed-bytes wrappers
// (Reference, BatchID, EthAddress, …) for length-validated identifiers.
//
// # Quickstart
//
// Connect to a local node, buy a postage batch, upload a few bytes:
//
//	import (
//	    "context"
//	    "log"
//	    "strings"
//
//	    bee "github.com/ethswarm-tools/bee-go"
//	    "github.com/ethswarm-tools/bee-go/pkg/swarm"
//	)
//
//	func run() {
//	    c, err := bee.NewClient("http://localhost:1633")
//	    if err != nil { log.Fatal(err) }
//
//	    ctx := context.Background()
//	    if ok, _ := c.Debug.Health(ctx); !ok {
//	        log.Fatal("bee node not healthy")
//	    }
//
//	    size, _ := swarm.SizeFromGigabytes(1)
//	    batchID, err := c.BuyStorage(ctx, size, swarm.DurationFromDays(30), nil)
//	    if err != nil { log.Fatal(err) }
//
//	    res, err := c.File.UploadData(ctx, batchID, strings.NewReader("Hello Swarm!"), nil)
//	    if err != nil { log.Fatal(err) }
//	    log.Printf("uploaded reference: %s", res.Reference.Hex())
//	}
//
// # Package layout
//
// bee-go is a sub-service client: the top-level [Client] exposes one
// sub-service per Bee API domain. This is a deliberate Go idiom — it
// keeps each domain's surface focused, allows compiler-checked imports
// of just what callers need, and avoids a single 100-method God object.
// (bee-js uses one flat Bee class because TypeScript has no equivalent
// of the import-pruning Go gives us for free.)
//
//   - [pkg/swarm]    typed bytes, token math (BZZ, DAI), Duration, Size, BMT
//     chunk addressing, SOC creation, GSOC mining, typed errors
//     (BeeError, BeeArgumentError, BeeResponseError), CheckResponse.
//   - [pkg/api]      core HTTP client + shared options/headers; pin, tag,
//     stewardship, grantee, envelope endpoints.
//   - [pkg/file]     data, file, chunk, SOC, feed and collection
//     uploads/downloads; FeedReader/FeedWriter; offline HashDirectory.
//   - [pkg/postage]  postage batch CRUD + stamp math (GetStampCost,
//     GetStampDuration, GetAmountForDuration, GetDepthForSize); offline
//     Stamper.
//   - [pkg/debug]    node info, peers, topology, balances, settlements,
//     chequebook, stake, transactions, redistribution.
//   - [pkg/pss]      PSS send/subscribe/receive over WebSockets.
//   - [pkg/gsoc]     Generic Single-Owner Chunk send/subscribe.
//   - [pkg/manifest] Mantaray trie + v0.2 wire format.
//
// Top-level helpers that span multiple sub-services — [Client.BuyStorage],
// [Client.ExtendStorage], [Client.GetStorageCost] and friends — live on
// [Client] itself.
//
// # Dev mode
//
// Use [NewDevClient] when targeting a "bee dev" node. The returned
// [DevClient] has the same surface as Client but the chain-state,
// chequebook, settlement, postage purchase and stake endpoints will
// return a *swarm.BeeResponseError with status 404.
//
// # Bee version compatibility
//
// This client targets Bee 2.7.1 / API version 7.4.1 (the values pinned
// in [pkg/debug.SupportedBeeVersionExact] and
// [pkg/debug.SupportedAPIVersion]). Use [pkg/debug.Service.IsSupportedExactVersion]
// for a strict match or [pkg/debug.Service.IsSupportedAPIVersion] for a
// major-version-compatible check at startup. Older / newer Bee
// instances usually work — unknown response fields are ignored — but
// new endpoints will return 404 and breaking wire-format changes will
// surface as JSON parse errors.
//
// # Authentication, timeouts, and proxies
//
// [Client] uses [http.DefaultClient] by default, which has no request
// timeout, no auth, and inherits proxy settings from the standard
// HTTP_PROXY / HTTPS_PROXY environment variables. For anything beyond a
// trusted local node, pass a configured [http.Client] via [WithHTTPClient]:
//
//	tr := &authTransport{token: os.Getenv("BEE_TOKEN"), base: http.DefaultTransport}
//	httpc := &http.Client{Transport: tr, Timeout: 30 * time.Second}
//	c, _ := bee.NewClient("https://bee.example.com", bee.WithHTTPClient(httpc))
//
// Where `authTransport` is a [http.RoundTripper] that adds an
// `Authorization: Bearer …` header. Bee gates `/stamps`, `/chequebook`,
// `/stake`, `/transactions`, and the operator endpoints behind tokens
// in production deployments.
//
// No automatic retries are performed. If you need transport-level
// retry, wrap the [http.RoundTripper] (e.g. with
// hashicorp/go-retryablehttp) before passing it to [WithHTTPClient].
//
// # Concurrency
//
// *[Client] and the sub-services it owns are safe for concurrent use
// from multiple goroutines. Construct one [Client] per Bee node URL and
// share it across your program — the underlying *[http.Client] manages
// its own connection pool. Sub-services hold pointers back to the same
// HTTP client, so per-Client tweaks (timeout, transport, auth) apply
// uniformly.
//
// # Cancellation
//
// Every endpoint takes a [context.Context] as its first argument.
// Cancelling the context aborts the in-flight HTTP request. For uploads,
// chunks already accepted by the local Bee node may remain in the local
// reserve but the upload is not committed (no manifest reference is
// returned). [pkg/file.Service.StreamDirectory] and
// [pkg/file.Service.StreamCollectionEntries] upload chunk-by-chunk and
// can leave orphan chunks if cancelled mid-stream.
//
// # Streaming vs. buffered transfers
//
// Downloads stream by default: [pkg/file.Service.DownloadData] and
// [pkg/file.Service.DownloadFile] return an [io.ReadCloser] backed by
// the live HTTP body. Drain it with [io.Copy] for large payloads;
// [io.ReadAll] buffers everything in memory and will OOM on multi-GB
// references.
//
// Uploads accept an [io.Reader] and stream the body to Bee. The
// streaming chunk-by-chunk variants ([pkg/file.Service.StreamDirectory]
// and friends) bound peak memory at the BMT chunk size (4 KiB × 128
// branches) regardless of file size; the tar-based
// [pkg/file.Service.UploadCollection] keeps the tar stream itself in
// memory.
//
// # Errors and retryability
//
// Every endpoint returns either nil or a *swarm.BeeError,
// *swarm.BeeArgumentError, or *swarm.BeeResponseError. Inspect with
// errors.As, or use the helper [pkg/swarm.IsBeeResponseError]:
//
//	if rerr, ok := swarm.IsBeeResponseError(err); ok {
//	    log.Printf("bee returned %d %s for %s %s",
//	        rerr.Status, rerr.StatusText, rerr.Method, rerr.URL)
//	}
//
// As a rule of thumb: 5xx responses and transport errors (DNS,
// connection refused, EOF) are retry candidates with backoff; 4xx
// responses are caller bugs (invalid batch ID, depth out of range,
// immutable-flag mismatch) and re-issuing the same request will fail
// the same way. POST /bytes uploads are idempotent for the same
// (data, batchID) tuple — Bee returns the same content reference.
//
// # Observability
//
// bee-go does not emit logs, metrics, or traces of its own. To
// instrument, wrap the [http.RoundTripper] passed to [WithHTTPClient]
// — that's the seam for request-level spans, latency histograms, and
// audit logs. Bee's own [pkg/debug.Service.GetLoggers] /
// [pkg/debug.Service.SetLoggerVerbosity] surface controls server-side
// verbosity at runtime.
//
// # Testing
//
// Point [NewClient] at an [net/http/httptest.Server] to test code that
// calls bee-go without running a real Bee:
//
//	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    w.Header().Set("Content-Type", "application/json")
//	    w.Write([]byte(`{"status":"ok","version":"2.7.1","apiVersion":"7.4.1"}`))
//	}))
//	defer srv.Close()
//	c, _ := bee.NewClient(srv.URL)
//	healthy, _ := c.Debug.Health(context.Background())
//
// # Common pitfalls
//
//   - A freshly-purchased postage batch is not usable for ~2-3 minutes
//     on Sepolia (block time × N confirmations). Polling
//     [pkg/postage.PostageBatch.Usable] before uploading avoids the
//     422 "stamp not usable" error.
//   - Dilute is one-way: depth can only grow, never shrink. Plan size
//     budgets accordingly.
//   - [pkg/swarm.ReferenceFromHex] accepts both 32-byte (plain) and
//     64-byte (encrypted) references — passing an encrypted reference
//     to a plain download will silently return garbage. Match the
//     reference type to how it was uploaded.
//   - Feed updates require the same (topic, signer) pair every time.
//     A new signer creates a new feed, not an update.
//   - On a `bee dev` node, all chain / chequebook / stake endpoints
//     return 404 — see [DevClient].
//
// # Go version
//
// bee-go requires Go 1.25 or newer (see go.mod).
//
// # Examples
//
// Runnable programs live under examples/ in the source tree:
// basic-usage (health + node info), buy-batch (postage purchase),
// upload-picture / download-picture (file round-trip), status
// (chain/reserve/redistribution state), integration-check (live-Bee
// soak). See the README for a bee-js → bee-go cheat sheet.
//
// [bee-js]: https://github.com/ethersphere/bee-js
package bee
