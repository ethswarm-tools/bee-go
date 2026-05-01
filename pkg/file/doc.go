// Package file implements every "data goes in or out of Bee" endpoint:
// raw bytes (/bytes), files (/bzz), chunks (/chunks), single-owner
// chunks (/soc), feeds (/feeds), and tar-packed collections (/bzz on a
// directory).
//
// Get a [*Service] handle from [github.com/ethswarm-tools/bee-go.Client.File].
//
// Headline pieces:
//
//   - Upload / download primitives — UploadData, DownloadData, UploadFile,
//     DownloadFile, UploadChunk, DownloadChunk, UploadSOC, ProbeData.
//   - Collections — UploadCollection (filesystem walk + tar), in-memory
//     UploadCollectionEntries, offline HashDirectory / HashCollectionEntries
//     (compute the manifest reference without uploading), and
//     StreamDirectory / StreamCollectionEntries for chunk-by-chunk uploads
//     with a per-chunk progress callback.
//   - Feeds — MakeFeedReader, MakeFeedWriter, MakeFeedIdentifier,
//     FeedUpdateChunkReference, IsFeedRetrievable,
//     AreAllSequentialFeedsUpdateRetrievable, FetchLatestFeedUpdate.
//   - Single-owner chunks — MakeSOCReader, MakeSOCWriter wrappers around
//     [pkg/swarm.MakeSingleOwnerChunk].
//
// Mirrors bee-js's Bee.uploadData / uploadFile / uploadFiles /
// uploadFilesFromDirectory / streamDirectory / uploadChunk / uploadSOC /
// downloadData / downloadFile / downloadChunk / makeFeedReader /
// makeFeedWriter / fetchLatestFeedUpdate / makeSOCReader / makeSOCWriter
// fan-out.
//
// # Streaming vs. buffered transfers
//
// Downloads return [io.ReadCloser] backed by the live HTTP body. Drain
// them with [io.Copy] for large payloads — [io.ReadAll] buffers the
// full reference in memory and will OOM on multi-GB downloads. Always
// Close the returned reader.
//
// Uploads accept [io.Reader] and stream the body to Bee. The
// chunk-by-chunk variants ([Service.StreamDirectory] and
// [Service.StreamCollectionEntries]) bound peak memory at the BMT
// chunk size regardless of file size and emit a per-chunk progress
// callback; the tar-based [Service.UploadCollection] keeps the tar
// stream itself in memory while it is being assembled.
//
// # Cancellation
//
// Cancelling the [context.Context] aborts the in-flight HTTP request.
// For [Service.StreamDirectory] / [Service.StreamCollectionEntries],
// chunks already accepted by the local Bee node remain in the local
// reserve but the manifest is not finalized — the resulting orphan
// chunks are eventually pruned but cost reserve space until then.
package file
