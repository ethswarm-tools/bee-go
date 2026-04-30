package postage

import (
	"github.com/ethersphere/bee-go/pkg/swarm"
)

// MarshaledStampLength is the byte length of a marshaled stamp:
// 32 (batchID) + 8 (index) + 8 (timestamp) + 65 (signature).
const MarshaledStampLength = 32 + 8 + 8 + 65

// MarshalStamp concatenates the stamp components into the wire format
// Bee expects when a stamp travels alongside a chunk: batchID || index
// || timestamp || signature (113 bytes).
//
// Mirrors bee-js marshalStamp (stamps.ts:181). Returns a typed argument
// error if any input has the wrong length, so callers can route the
// failure through the BeeArgumentError surface.
func MarshalStamp(batchID swarm.BatchID, index, timestamp []byte, signature swarm.Signature) ([]byte, error) {
	if len(index) != 8 {
		return nil, swarm.NewBeeArgumentError("invalid index length", len(index))
	}
	if len(timestamp) != 8 {
		return nil, swarm.NewBeeArgumentError("invalid timestamp length", len(timestamp))
	}
	if len(signature.Raw()) != swarm.SignatureLength {
		return nil, swarm.NewBeeArgumentError("invalid signature length", len(signature.Raw()))
	}
	out := make([]byte, 0, MarshaledStampLength)
	out = append(out, batchID.Raw()...)
	out = append(out, index...)
	out = append(out, timestamp...)
	out = append(out, signature.Raw()...)
	return out, nil
}

// ConvertEnvelopeToMarshaledStamp marshals a stamper Envelope into the
// wire format. Convenience wrapper around MarshalStamp for callers that
// already hold a structured Envelope (typically the return value of
// Stamper.Stamp).
//
// Mirrors bee-js convertEnvelopeToMarshaledStamp (stamps.ts:177).
func ConvertEnvelopeToMarshaledStamp(env Envelope) ([]byte, error) {
	return MarshalStamp(env.BatchID, env.Index, env.Timestamp, env.Signature)
}
