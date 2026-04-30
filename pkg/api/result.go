package api

import (
	"net/http"
	"strconv"

	"github.com/ethersphere/bee-go/pkg/swarm"
)

// UploadResult is the standardized return shape for every upload endpoint.
// Mirrors bee-js's UploadResult. TagUid is 0 when Bee did not return a
// swarm-tag header. HistoryAddress is nil when no ACT history was created.
type UploadResult struct {
	Reference      swarm.Reference
	TagUid         uint32
	HistoryAddress *swarm.Reference
}

// ReadUploadResult parses an UploadResult from an HTTP response. The
// reference is taken from the JSON body (already decoded into refHex by the
// caller), the tag UID and history address come from response headers.
func ReadUploadResult(refHex string, headers http.Header) (UploadResult, error) {
	ref, err := swarm.ReferenceFromHex(refHex)
	if err != nil {
		return UploadResult{}, err
	}
	res := UploadResult{Reference: ref}
	if tag := headers.Get("Swarm-Tag"); tag != "" {
		if v, err := strconv.ParseUint(tag, 10, 32); err == nil {
			res.TagUid = uint32(v)
		}
	}
	if hist := headers.Get("Swarm-Act-History-Address"); hist != "" {
		if r, err := swarm.ReferenceFromHex(hist); err == nil {
			res.HistoryAddress = &r
		}
	}
	return res, nil
}

// FileHeaders is the parsed form of the response headers Bee returns when
// downloading a file (Content-Disposition / swarm-tag-uid / Content-Type).
type FileHeaders struct {
	Name        string
	TagUid      uint32
	ContentType string
}

// ParseFileHeaders extracts the file metadata Bee places on download
// responses. Missing or malformed headers fall through silently — the
// returned FileHeaders zero values mirror bee-js's `undefined` semantics.
func ParseFileHeaders(headers http.Header) FileHeaders {
	out := FileHeaders{
		ContentType: headers.Get("Content-Type"),
	}
	if cd := headers.Get("Content-Disposition"); cd != "" {
		out.Name = parseContentDispositionFilename(cd)
	}
	if tag := headers.Get("Swarm-Tag-Uid"); tag != "" {
		if v, err := strconv.ParseUint(tag, 10, 32); err == nil {
			out.TagUid = uint32(v)
		}
	}
	return out
}

// parseContentDispositionFilename extracts the filename from a
// Content-Disposition header. Returns "" if no filename is present. Mirrors
// bee-js's regex-based parser.
func parseContentDispositionFilename(header string) string {
	// We look for `filename=...` or `filename*=UTF-8''...`. Strip quotes and
	// trailing semicolons. This intentionally permissive parser matches what
	// bee-js does — Bee only emits a small set of well-formed values.
	for _, part := range splitSemicolons(header) {
		part = trimSpace(part)
		const key = "filename="
		if !startsWithFold(part, key) && !startsWithFold(part, "filename*=") {
			continue
		}
		eq := indexByte(part, '=')
		v := part[eq+1:]
		// filename*=UTF-8''actual-name
		if i := indexString(v, "''"); i >= 0 {
			v = v[i+2:]
		}
		v = trimQuotes(v)
		return v
	}
	return ""
}

func splitSemicolons(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ';' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func trimQuotes(s string) string {
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"') {
		return s[1 : len(s)-1]
	}
	return s
}

func startsWithFold(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		a := s[i]
		b := prefix[i]
		if a >= 'A' && a <= 'Z' {
			a += 'a' - 'A'
		}
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func indexString(s, sub string) int {
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
