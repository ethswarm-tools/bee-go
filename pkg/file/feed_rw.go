package file

import (
	"context"

	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

// FeedReader reads updates from a feed identified by (Owner, Topic).
// Mirrors bee-js FeedReader. Construct via Service.MakeFeedReader.
type FeedReader struct {
	service *Service
	Owner   swarm.EthAddress
	Topic   swarm.Topic
}

// MakeFeedReader returns a FeedReader bound to (owner, topic).
func (s *Service) MakeFeedReader(owner swarm.EthAddress, topic swarm.Topic) *FeedReader {
	return &FeedReader{service: s, Owner: owner, Topic: topic}
}

// Download returns the latest feed update.
func (r *FeedReader) Download(ctx context.Context) (FeedUpdate, error) {
	return r.service.FetchLatestFeedUpdate(ctx, r.Owner, r.Topic)
}

// DownloadReference returns the latest feed update interpreted as
// `timestamp(8) || reference(32)` and gives back the embedded reference.
// Returns an error if the payload is not the expected length.
//
// Use this when the feed stores references (the common case for mutable
// content) — the writer side is UpdateFeedWithReference.
func (r *FeedReader) DownloadReference(ctx context.Context) (swarm.Reference, FeedUpdate, error) {
	upd, err := r.Download(ctx)
	if err != nil {
		return swarm.Reference{}, FeedUpdate{}, err
	}
	if len(upd.Payload) < 8+swarm.ReferenceLength {
		return swarm.Reference{}, upd, swarm.NewBeeError("feed payload too short for reference")
	}
	ref, err := swarm.NewReference(upd.Payload[8 : 8+swarm.ReferenceLength])
	if err != nil {
		return swarm.Reference{}, upd, err
	}
	return ref, upd, nil
}

// FeedWriter is a FeedReader plus update methods. Construct via
// Service.MakeFeedWriter, which derives the owner from the signer.
type FeedWriter struct {
	*FeedReader
	signer swarm.PrivateKey
}

// MakeFeedWriter returns a FeedWriter for (signer, topic). The owner is
// derived from the signer's public key.
func (s *Service) MakeFeedWriter(signer swarm.PrivateKey, topic swarm.Topic) *FeedWriter {
	owner := signer.PublicKey().Address()
	return &FeedWriter{
		FeedReader: s.MakeFeedReader(owner, topic),
		signer:     signer,
	}
}

// Upload writes `data` to the next feed index, wrapped as
// `timestamp(8) || data`. Mirrors bee-js FeedWriter.uploadPayload.
func (w *FeedWriter) Upload(ctx context.Context, batchID swarm.BatchID, data []byte) (api.UploadResult, error) {
	return w.service.UpdateFeed(ctx, batchID, w.signer, w.Topic, data)
}

// UploadReference writes a Reference to the next feed index. Mirrors
// bee-js FeedWriter.upload / .uploadReference.
func (w *FeedWriter) UploadReference(ctx context.Context, batchID swarm.BatchID, ref swarm.Reference) (api.UploadResult, error) {
	return w.service.UpdateFeedWithReference(ctx, batchID, w.signer, w.Topic, ref, nil)
}

// UploadAtIndex writes `data` at a specific feed index, bypassing
// FindNextIndex. Useful for replaying or seeding feeds at known indexes.
func (w *FeedWriter) UploadAtIndex(ctx context.Context, batchID swarm.BatchID, index uint64, data []byte) (api.UploadResult, error) {
	return w.service.UpdateFeedWithIndex(ctx, batchID, w.signer, w.Topic, index, data)
}
