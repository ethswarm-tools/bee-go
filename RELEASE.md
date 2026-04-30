# Releasing bee-go

## Versioning

bee-go follows [Semantic Versioning](https://semver.org/) once `v1.0.0` is
tagged. Pre-1.0 releases (`v0.x.y`) may break the public API on a minor
version bump; patch bumps stay backwards-compatible.

The public API surface is everything exported from the top-level `bee-go`
module and its `pkg/...` subpackages. Internal helpers and `examples/` are
not covered by the compatibility promise.

## Cutting a release

1. **Update CHANGELOG.md.** Move the contents of `[Unreleased]` to a new
   `[X.Y.Z] — YYYY-MM-DD` section. Leave an empty `[Unreleased]` block
   in place for the next cycle.
2. **Verify the tree is clean.**
   ```sh
   go vet ./...
   go test -race -count=1 ./...
   golangci-lint run
   ```
3. **Run the live integration check** against a Bee node you control
   (mainnet or Sepolia). Reuse a usable batch via `BEE_BATCH_ID` to avoid
   the multi-minute Sepolia wait:
   ```sh
   BEE_URL=http://localhost:1633 BEE_BATCH_ID=<hex> \
       go run ./examples/integration-check
   ```
   Expect ≥ 53 / 54 passing. Investigate any new failures before tagging.
4. **Bump `SupportedBeeVersionExact`** in `pkg/debug/node.go` if this
   release was tested against a newer Bee build than the previous tag.
5. **Commit** the CHANGELOG move + any version bumps as a single
   `release: vX.Y.Z` commit.
6. **Tag and push.**
   ```sh
   git tag -a vX.Y.Z -m "vX.Y.Z"
   git push origin main vX.Y.Z
   ```
7. **Create a GitHub release** from the tag, copying the relevant
   CHANGELOG section into the release notes.

## What goes into CHANGELOG.md

- **Added** — new exported surface.
- **Changed** — observable behavior changes. Prefix with `**BREAKING:**`
  if the change requires caller updates; pre-1.0 these are allowed on
  minor bumps but must be called out.
- **Fixed** — bug fixes, especially ones surfaced by the live
  integration check or downstream users.
- **Removed** — deprecated surface that is finally gone.

Keep entries terse but specific — name the symbol, say what changed,
link to a PR or issue if there is one.
