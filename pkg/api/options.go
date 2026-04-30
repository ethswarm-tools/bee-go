package api

import (
	"fmt"
	"net/http"

	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

// RedundancyLevel is the data redundancy level applied when uploading.
// Mirrors bee-js's RedundancyLevel enum.
type RedundancyLevel int

const (
	RedundancyLevelOff      RedundancyLevel = 0
	RedundancyLevelMedium   RedundancyLevel = 1
	RedundancyLevelStrong   RedundancyLevel = 2
	RedundancyLevelInsane   RedundancyLevel = 3
	RedundancyLevelParanoid RedundancyLevel = 4
)

// RedundancyStrategy is the chunk-prefetch policy used when downloading
// erasure-coded data. Mirrors bee-js's RedundancyStrategy enum.
type RedundancyStrategy int

const (
	RedundancyStrategyNone RedundancyStrategy = 0
	RedundancyStrategyData RedundancyStrategy = 1
	RedundancyStrategyProx RedundancyStrategy = 2
	RedundancyStrategyRace RedundancyStrategy = 3
)

// UploadOptions is the base set of options accepted by every upload endpoint.
// Mirrors bee-js's UploadOptions.
//
// Pointer fields (*bool) distinguish "unset" from "explicitly false". Bee
// reads any of these only if the corresponding header is present; we omit
// the header entirely when the pointer is nil.
type UploadOptions struct {
	// Act, when true, instructs Bee to create an Access Control Trie (ACT)
	// for the uploaded data. The history address is returned in the
	// swarm-act-history-address response header.
	Act *bool

	// ActHistoryAddress extends an existing ACT history when re-uploading
	// updated content under the same access policy.
	ActHistoryAddress *swarm.Reference

	// Pin keeps a local copy of the uploaded data on the Bee node so it can
	// be re-uploaded if it disappears from the network.
	Pin *bool

	// Encrypt instructs Bee to encrypt the chunks; the returned reference
	// includes the decryption key (64-byte reference instead of 32).
	Encrypt *bool

	// Tag attaches an existing tag UID to the upload to track sync progress.
	Tag uint32

	// Deferred toggles between "client waits for full sync" (false) and
	// "Bee accepts upload then syncs in the background" (true). Default in
	// Bee is true.
	Deferred *bool
}

// RedundantUploadOptions adds the redundancy level applied to data uploads.
type RedundantUploadOptions struct {
	UploadOptions
	RedundancyLevel RedundancyLevel
}

// FileUploadOptions adds the file-specific knobs used by POST /bzz uploads.
type FileUploadOptions struct {
	UploadOptions
	// Size sets Content-Length explicitly. Required when uploading from an
	// io.Reader of unknown length.
	Size int64
	// ContentType becomes the file's reported MIME type.
	ContentType string
	// RedundancyLevel adds erasure coding to the upload.
	RedundancyLevel RedundancyLevel
}

// CollectionUploadOptions adds the directory-specific knobs used by tar
// uploads on POST /bzz.
type CollectionUploadOptions struct {
	UploadOptions
	// IndexDocument is served when the collection root is requested
	// (e.g. "index.html").
	IndexDocument string
	// ErrorDocument is served when a path inside the collection is missing.
	ErrorDocument string
	// RedundancyLevel adds erasure coding to the upload.
	RedundancyLevel RedundancyLevel
}

// DownloadOptions controls retrieval behaviour. All fields are optional;
// passing nil to a download method keeps Bee defaults.
type DownloadOptions struct {
	// RedundancyStrategy picks a chunk-prefetch policy for erasure coded
	// data.
	RedundancyStrategy *RedundancyStrategy

	// Fallback toggles whether retrieve strategies cascade. Default true.
	Fallback *bool

	// TimeoutMs is the per-chunk retrieval timeout (not the whole download).
	TimeoutMs int

	// ActPublisher is the public key of the ACT publisher when reading
	// access-controlled data.
	ActPublisher *swarm.PublicKey

	// ActHistoryAddress is the history root used to resolve permissions at
	// ActTimestamp.
	ActHistoryAddress *swarm.Reference

	// ActTimestamp is the Unix timestamp at which to evaluate ACT
	// permissions. 0 means "now".
	ActTimestamp int64
}

// PostageBatchOptions covers query-string knobs accepted by stamp creation
// (label, immutable, gas overrides). Mirrors bee-js PostageBatchOptions.
type PostageBatchOptions struct {
	Label     string
	Immutable *bool
	GasPrice  string
	GasLimit  string
}

// ============================================================================
// Header preparation
// ============================================================================

// PrepareUploadHeaders writes every applicable upload header onto req. The
// batch is required (Bee rejects uploads without a stamp).
func PrepareUploadHeaders(req *http.Request, batchID swarm.BatchID, opts *UploadOptions) {
	req.Header.Set("Swarm-Postage-Batch-Id", batchID.Hex())
	if opts == nil {
		return
	}
	applyUploadOptions(req, opts)
}

// PrepareRedundantUploadHeaders is PrepareUploadHeaders + Swarm-Redundancy-Level.
func PrepareRedundantUploadHeaders(req *http.Request, batchID swarm.BatchID, opts *RedundantUploadOptions) {
	if opts == nil {
		PrepareUploadHeaders(req, batchID, nil)
		return
	}
	PrepareUploadHeaders(req, batchID, &opts.UploadOptions)
	if opts.RedundancyLevel > RedundancyLevelOff {
		req.Header.Set("Swarm-Redundancy-Level", fmt.Sprintf("%d", opts.RedundancyLevel))
	}
}

// PrepareFileUploadHeaders prepares the headers for a POST /bzz file upload.
// It also overrides Content-Type / Content-Length if FileUploadOptions sets
// them.
func PrepareFileUploadHeaders(req *http.Request, batchID swarm.BatchID, opts *FileUploadOptions) {
	if opts == nil {
		PrepareUploadHeaders(req, batchID, nil)
		return
	}
	PrepareUploadHeaders(req, batchID, &opts.UploadOptions)
	if opts.Size > 0 {
		req.Header.Set("Content-Length", fmt.Sprintf("%d", opts.Size))
	}
	if opts.ContentType != "" {
		req.Header.Set("Content-Type", opts.ContentType)
	}
	if opts.RedundancyLevel > RedundancyLevelOff {
		req.Header.Set("Swarm-Redundancy-Level", fmt.Sprintf("%d", opts.RedundancyLevel))
	}
}

// PrepareCollectionUploadHeaders prepares the headers for a tar /bzz upload.
func PrepareCollectionUploadHeaders(req *http.Request, batchID swarm.BatchID, opts *CollectionUploadOptions) {
	if opts == nil {
		PrepareUploadHeaders(req, batchID, nil)
		return
	}
	PrepareUploadHeaders(req, batchID, &opts.UploadOptions)
	if opts.IndexDocument != "" {
		req.Header.Set("Swarm-Index-Document", opts.IndexDocument)
	}
	if opts.ErrorDocument != "" {
		req.Header.Set("Swarm-Error-Document", opts.ErrorDocument)
	}
	if opts.RedundancyLevel > RedundancyLevelOff {
		req.Header.Set("Swarm-Redundancy-Level", fmt.Sprintf("%d", opts.RedundancyLevel))
	}
}

// PrepareDownloadHeaders writes every applicable download header onto req.
// nil opts is a no-op (Bee defaults are used).
func PrepareDownloadHeaders(req *http.Request, opts *DownloadOptions) {
	if opts == nil {
		return
	}
	if opts.RedundancyStrategy != nil {
		req.Header.Set("Swarm-Redundancy-Strategy", fmt.Sprintf("%d", *opts.RedundancyStrategy))
	}
	if opts.Fallback != nil {
		req.Header.Set("Swarm-Redundancy-Fallback-Mode", boolStr(*opts.Fallback))
	}
	if opts.TimeoutMs > 0 {
		req.Header.Set("Swarm-Chunk-Retrieval-Timeout", fmt.Sprintf("%d", opts.TimeoutMs))
	}
	applyACTDownload(req, opts)
}

// applyUploadOptions writes the UploadOptions headers; assumes batchID is
// already set.
func applyUploadOptions(req *http.Request, opts *UploadOptions) {
	if opts.Pin != nil {
		req.Header.Set("Swarm-Pin", boolStr(*opts.Pin))
	}
	if opts.Encrypt != nil {
		req.Header.Set("Swarm-Encrypt", boolStr(*opts.Encrypt))
	}
	if opts.Tag > 0 {
		req.Header.Set("Swarm-Tag", fmt.Sprintf("%d", opts.Tag))
	}
	if opts.Deferred != nil {
		req.Header.Set("Swarm-Deferred-Upload", boolStr(*opts.Deferred))
	}
	if opts.Act != nil {
		req.Header.Set("Swarm-Act", boolStr(*opts.Act))
	}
	if opts.ActHistoryAddress != nil {
		req.Header.Set("Swarm-Act-History-Address", opts.ActHistoryAddress.Hex())
	}
}

// applyACTDownload writes the act-* download headers. Setting any of
// publisher / history / timestamp implicitly turns Swarm-Act on.
func applyACTDownload(req *http.Request, opts *DownloadOptions) {
	any := false
	if opts.ActPublisher != nil {
		hex, err := opts.ActPublisher.CompressedHex()
		if err == nil {
			req.Header.Set("Swarm-Act-Publisher", hex)
			any = true
		}
	}
	if opts.ActHistoryAddress != nil {
		req.Header.Set("Swarm-Act-History-Address", opts.ActHistoryAddress.Hex())
		any = true
	}
	if opts.ActTimestamp > 0 {
		req.Header.Set("Swarm-Act-Timestamp", fmt.Sprintf("%d", opts.ActTimestamp))
		any = true
	}
	if any {
		req.Header.Set("Swarm-Act", "true")
	}
}

// BoolPtr is a convenience for setting *bool fields on options structs:
//
//	opts := &api.UploadOptions{Pin: api.BoolPtr(true)}
func BoolPtr(b bool) *bool { return &b }

// RedundancyStrategyPtr is the analogous convenience for *RedundancyStrategy.
func RedundancyStrategyPtr(s RedundancyStrategy) *RedundancyStrategy { return &s }

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// ApplyToRequest is preserved for backwards compatibility with previously
// inlined call sites; new code should use PrepareUploadHeaders instead. It
// only sets the four legacy fields.
//
// Deprecated: use PrepareUploadHeaders or one of the typed Prepare* helpers.
func (o *UploadOptions) ApplyToRequest(req *http.Request) {
	if o == nil {
		return
	}
	applyUploadOptions(req, o)
}
