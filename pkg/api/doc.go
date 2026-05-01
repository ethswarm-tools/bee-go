// Package api covers the core HTTP service used by every other bee-go
// sub-package, plus the cross-cutting endpoints that don't fit neatly
// into a single domain: pin, tag, stewardship, grantee, envelope, and
// "is reference retrievable?" checks.
//
// It also defines the upload / download option structs that bee-go's
// upload methods accept:
//
//   - [UploadOptions] — base options (pin, encrypt, tag, deferred, ACT)
//   - [RedundantUploadOptions] — adds redundancy level
//   - [FileUploadOptions] — adds size + content-type for /bzz uploads
//   - [CollectionUploadOptions] — adds index/error documents for tar uploads
//   - [DownloadOptions] — redundancy-strategy + ACT-grantee fields
//   - [PostageBatchOptions] — label / immutable / gas-price / gas-limit
//
// Mirrors bee-js's UploadOptions / DownloadOptions / PostageBatchOptions
// fan-out and bee-js's pin / tag / stewardship / grantee endpoints.
package api
