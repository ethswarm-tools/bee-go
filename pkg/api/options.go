package api

import (
	"fmt"
	"net/http"
)

// RedundancyLevel represents the data redundancy level.
type RedundancyLevel int

const (
	RedundancyLevelOff      RedundancyLevel = 0
	RedundancyLevelMedium   RedundancyLevel = 1
	RedundancyLevelStrong   RedundancyLevel = 2
	RedundancyLevelInsane   RedundancyLevel = 3
	RedundancyLevelParanoid RedundancyLevel = 4
)

// UploadOptions represents optional parameters for uploads.
type UploadOptions struct {
	Pin        bool
	Encrypt    bool
	Tag        uint32
	Redundancy RedundancyLevel
	Deferred   *bool // Optional pointer to distinguish true/false/nil
}

// ApplyToRequest applies the options to the HTTP request headers.
func (o *UploadOptions) ApplyToRequest(req *http.Request) {
	if o == nil {
		return
	}
	if o.Pin {
		req.Header.Set("Swarm-Pin", "true")
	}
	if o.Encrypt {
		req.Header.Set("Swarm-Encrypt", "true")
	}
	if o.Tag > 0 {
		req.Header.Set("Swarm-Tag", fmt.Sprintf("%d", o.Tag))
	}
	if o.Redundancy > RedundancyLevelOff {
		req.Header.Set("Swarm-Redundancy-Level", fmt.Sprintf("%d", o.Redundancy))
	}
	if o.Deferred != nil {
		req.Header.Set("Swarm-Deferred-Upload", fmt.Sprintf("%t", *o.Deferred))
	}
}
