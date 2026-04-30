// integration-check exercises bee-go against a real Bee node.
//
// Run against a node that has BZZ + native funds; the program buys a
// small postage batch and a few stamps will be consumed. It is safe
// to re-run — every artifact is created fresh each invocation.
//
//	go run ./examples/integration-check
//
// Override the URL with BEE_URL env if not http://localhost:1633.
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/file"
	"github.com/ethswarm-tools/bee-go/pkg/gsoc"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

var (
	pass int
	fail int
)

func main() {
	url := os.Getenv("BEE_URL")
	if url == "" {
		url = "http://localhost:1633"
	}
	fmt.Printf("Bee URL: %s\n\n", url)

	c, err := bee.NewClient(url)
	if err != nil {
		fatalf("NewClient: %v", err)
	}
	ctx := context.Background()

	section("Read-only — connectivity & node info")
	check("IsConnected", func() error {
		if !c.Debug.IsConnected(ctx) {
			return fmt.Errorf("not connected")
		}
		return nil
	})
	check("CheckConnection", func() error { return c.Debug.CheckConnection(ctx) })
	check("GetHealth", func() error {
		h, err := c.Debug.GetHealth(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("    status=%s version=%s apiVersion=%s\n", h.Status, h.Version, h.APIVersion)
		return nil
	})
	check("GetVersions", func() error {
		v, err := c.Debug.GetVersions(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("    bee=%s api=%s | client supports bee=%s api=%s\n",
			v.BeeVersion, v.BeeAPIVersion, v.SupportedBeeVersion, v.SupportedBeeAPIVersion)
		return nil
	})
	check("IsSupportedAPIVersion", func() error {
		ok, err := c.Debug.IsSupportedAPIVersion(ctx)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("API major-version mismatch")
		}
		return nil
	})
	check("IsSupportedExactVersion", func() error {
		_, err := c.Debug.IsSupportedExactVersion(ctx)
		// We expect this to often be false on a moving testnet — only fail on error.
		return err
	})
	check("IsGateway", func() error {
		_, err := c.Debug.IsGateway(ctx)
		return err
	})
	check("Readiness", func() error {
		_, err := c.Debug.Readiness(ctx)
		return err
	})
	check("NodeInfo", func() error {
		ni, err := c.Debug.NodeInfo(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("    beeMode=%s chequebook=%v swap=%v\n", ni.BeeMode, ni.ChequebookEnabled, ni.SwapEnabled)
		return nil
	})
	check("Status", func() error {
		s, err := c.Debug.Status(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("    overlay=%s reserveSize=%d connectedPeers=%d isReachable=%v\n",
			s.Overlay, s.ReserveSize, s.ConnectedPeers, s.IsReachable)
		return nil
	})
	check("Addresses", func() error {
		_, err := c.Debug.Addresses(ctx)
		return err
	})
	check("Topology", func() error {
		t, err := c.Debug.Topology(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("    population=%d connected=%d depth=%d\n", t.Population, t.Connected, t.Depth)
		return nil
	})
	check("Peers", func() error {
		p, err := c.Debug.Peers(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("    %d peers\n", len(p.Peers))
		return nil
	})
	check("ChainState", func() error {
		s, err := c.Debug.ChainState(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("    chainTip=%d currentPrice=%d\n", s.ChainTip, s.CurrentPrice)
		return nil
	})
	check("ReserveState", func() error {
		_, err := c.Debug.ReserveState(ctx)
		return err
	})
	check("RedistributionState", func() error {
		_, err := c.Debug.RedistributionState(ctx)
		return err
	})

	section("Read-only — wallet, stake, accounting")
	check("GetWallet", func() error {
		w, err := c.Debug.GetWallet(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("    bzz=%s native=%s chainID=%d\n",
			w.BzzBalance.String(), w.NativeTokenBalance.String(), w.ChainID)
		return nil
	})
	check("GetChequebookBalance", func() error {
		_, err := c.Debug.GetChequebookBalance(ctx)
		return err
	})
	check("GetStake", func() error {
		s, err := c.Debug.GetStake(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("    staked=%s\n", s.String())
		return nil
	})
	check("GetWithdrawableStake", func() error {
		_, err := c.Debug.GetWithdrawableStake(ctx)
		return err
	})
	check("GetBalances", func() error {
		_, err := c.Debug.GetBalances(ctx)
		return err
	})
	check("Settlements", func() error {
		_, err := c.Debug.Settlements(ctx)
		return err
	})
	check("LastCheques", func() error {
		_, err := c.Debug.LastCheques(ctx)
		return err
	})
	check("GetAccounting", func() error {
		_, err := c.Debug.GetAccounting(ctx)
		return err
	})
	check("StatusPeers", func() error {
		_, err := c.Debug.StatusPeers(ctx)
		return err
	})
	check("StatusNeighborhoods", func() error {
		_, err := c.Debug.StatusNeighborhoods(ctx)
		return err
	})
	check("GetWelcomeMessage", func() error {
		_, err := c.Debug.GetWelcomeMessage(ctx)
		return err
	})
	check("GetLoggers", func() error {
		_, err := c.Debug.GetLoggers(ctx)
		return err
	})

	section("Read-only — postage")
	check("GetPostageBatches", func() error {
		b, err := c.Postage.GetPostageBatches(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("    %d owned batches\n", len(b))
		return nil
	})
	check("GetGlobalPostageBatches", func() error {
		b, err := c.Postage.GetGlobalPostageBatches(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("    %d global batches on chain\n", len(b))
		return nil
	})

	section("Tags — full lifecycle")
	var tagUID uint32
	check("CreateTag", func() error {
		tag, err := c.API.CreateTag(ctx)
		if err != nil {
			return err
		}
		tagUID = tag.UID
		fmt.Printf("    uid=%d\n", tagUID)
		return nil
	})
	check("GetTag", func() error {
		_, err := c.API.GetTag(ctx, tagUID)
		return err
	})
	check("RetrieveTag (alias)", func() error {
		_, err := c.API.RetrieveTag(ctx, tagUID)
		return err
	})
	check("ListTags", func() error {
		ts, err := c.API.ListTags(ctx, 0, 100)
		if err != nil {
			return err
		}
		fmt.Printf("    %d tags listed\n", len(ts))
		return nil
	})
	check("DeleteTag", func() error { return c.API.DeleteTag(ctx, tagUID) })

	section("Offline — content addressing (no node call)")
	entries := []file.CollectionEntry{
		{Path: "index.html", Data: []byte("<h1>integration check</h1>")},
		{Path: "data.bin", Data: bytes.Repeat([]byte{0x42}, swarm.ChunkSize+200)},
	}
	var hashedRoot swarm.Reference
	check("HashCollectionEntries", func() error {
		ref, err := file.HashCollectionEntries(entries)
		if err != nil {
			return err
		}
		hashedRoot = ref
		fmt.Printf("    root=%s\n", ref.Hex())
		return nil
	})
	check("MakeContentAddressedChunk", func() error {
		_, err := swarm.MakeContentAddressedChunk([]byte("hi"))
		return err
	})

	section("Postage — buying a small batch (or reusing BEE_BATCH_ID)")
	var batchID swarm.BatchID
	boughtBatch := false
	if existing := os.Getenv("BEE_BATCH_ID"); existing != "" {
		check("Reuse BEE_BATCH_ID", func() error {
			id, err := swarm.BatchIDFromHex(existing)
			if err != nil {
				return err
			}
			batchID = id
			fmt.Printf("    batchID=%s (from env)\n", id.Hex())
			return nil
		})
	} else {
		check("BuyStorage (depth 17, ~24h)", func() error {
			size, err := swarm.SizeFromKilobytes(1)
			if err != nil {
				return err
			}
			id, err := c.BuyStorage(ctx,
				size,
				swarm.DurationFromDays(1),
				&bee.StorageOptions{Network: bee.NetworkMainnet},
			)
			if err != nil {
				return err
			}
			batchID = id
			boughtBatch = true
			fmt.Printf("    batchID=%s\n", id.Hex())
			return nil
		})
	}
	batchUsable := false
	if !batchID.IsZero() {
		check("Wait until batch is usable (poll /stamps; Bee 400s the per-id endpoint while non-usable)", func() error {
			start := time.Now()
			deadline := start.Add(5 * time.Minute)
			for {
				batches, err := c.Postage.GetPostageBatches(ctx)
				if err != nil {
					return err
				}
				for _, b := range batches {
					if b.BatchID.Hex() == batchID.Hex() && b.Usable {
						fmt.Printf("    usable after %s, depth=%d ttl=%ds\n", time.Since(start).Round(time.Second), b.Depth, b.BatchTTL)
						batchUsable = true
						return nil
					}
				}
				if time.Now().After(deadline) {
					return fmt.Errorf("timeout after %s waiting for batch usable", time.Since(start).Round(time.Second))
				}
				time.Sleep(5 * time.Second)
			}
		})
		if batchUsable {
			check("GetPostageBatch (per-id)", func() error {
				b, err := c.Postage.GetPostageBatch(ctx, batchID)
				if err != nil {
					return err
				}
				fmt.Printf("    utilization=%d usable=%v\n", b.Utilization, b.Usable)
				return nil
			})
			check("GetPostageBatchBuckets", func() error {
				b, err := c.Postage.GetPostageBatchBuckets(ctx, batchID)
				if err != nil {
					return err
				}
				fmt.Printf("    depth=%d bucketDepth=%d buckets=%d\n", b.Depth, b.BucketDepth, len(b.Buckets))
				return nil
			})
		} else {
			skip("GetPostageBatch (per-id)", "batch never became usable")
			skip("GetPostageBatchBuckets", "batch never became usable")
		}
	} else {
		skip("Wait until batch is usable", "no batchID")
		skip("GetPostageBatch (per-id)", "no batchID")
		skip("GetPostageBatchBuckets", "no batchID")
	}

	section("Upload round-trip (UploadData → ProbeData → DownloadData → Pin/Unpin)")
	var dataRef swarm.Reference
	if batchUsable {
		payload := []byte("integration-check payload " + time.Now().UTC().Format(time.RFC3339Nano))
		check("UploadData", func() error {
			res, err := c.File.UploadData(ctx, batchID, bytes.NewReader(payload), nil)
			if err != nil {
				return err
			}
			dataRef = res.Reference
			fmt.Printf("    ref=%s\n", res.Reference.Hex())
			return nil
		})
		check("ProbeData", func() error {
			info, err := c.File.ProbeData(ctx, dataRef)
			if err != nil {
				return err
			}
			if info.ContentLength != int64(len(payload)) {
				return fmt.Errorf("ContentLength=%d want %d", info.ContentLength, len(payload))
			}
			fmt.Printf("    contentLength=%d\n", info.ContentLength)
			return nil
		})
		check("DownloadData (round-trip equality)", func() error {
			rc, err := c.File.DownloadData(ctx, dataRef, nil)
			if err != nil {
				return err
			}
			defer rc.Close()
			got, err := io.ReadAll(rc)
			if err != nil {
				return err
			}
			if !bytes.Equal(got, payload) {
				return fmt.Errorf("payload mismatch (%d vs %d bytes)", len(got), len(payload))
			}
			return nil
		})
		check("Pin", func() error { return c.API.Pin(ctx, dataRef) })
		check("GetPin", func() error {
			ok, err := c.API.GetPin(ctx, dataRef)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("not pinned after Pin")
			}
			return nil
		})
		check("ListPins", func() error {
			_, err := c.API.ListPins(ctx)
			return err
		})
		check("Unpin", func() error { return c.API.Unpin(ctx, dataRef) })
		check("IsRetrievable", func() error {
			_, err := c.API.IsRetrievable(ctx, dataRef)
			return err
		})
	} else {
		skip("UploadData / ProbeData / DownloadData / Pin / Unpin", "batch not usable")
	}

	section("SOC round-trip (MakeSOCWriter → MakeSOCReader)")
	if batchUsable {
		signer, _ := crypto.GenerateKey()
		w, err := c.File.MakeSOCWriter(signer)
		if err != nil {
			fatalf("MakeSOCWriter: %v", err)
		}
		topic := swarm.IdentifierFromString("integration-soc-" + time.Now().Format(time.RFC3339Nano))
		socPayload := []byte("hello from integration-check")

		check("SOCWriter.Upload", func() error {
			res, err := w.Upload(ctx, batchID, topic, socPayload, nil)
			if err != nil {
				return err
			}
			fmt.Printf("    socRef=%s\n", res.Reference.Hex())
			return nil
		})
		check("SOCReader.Download (equality)", func() error {
			got, err := w.Download(ctx, topic)
			if err != nil {
				return err
			}
			if !bytes.Equal(got.Payload, socPayload) {
				return fmt.Errorf("SOC payload mismatch")
			}
			return nil
		})
	} else {
		skip("SOC round-trip", "batch not usable")
	}

	section("Stream upload (StreamCollectionEntries → DownloadData of root via /bzz)")
	if batchUsable {
		var streamRoot swarm.Reference
		check("StreamCollectionEntries", func() error {
			res, err := c.File.StreamCollectionEntries(ctx, batchID, entries, nil)
			if err != nil {
				return err
			}
			streamRoot = res.Reference
			fmt.Printf("    manifestRef=%s (offline-hash=%s)\n", res.Reference.Hex(), hashedRoot.Hex())
			return nil
		})
		// The streamed manifest root must equal the offline-hashed root.
		check("Streamed root == offline hash", func() error {
			if streamRoot.Hex() != hashedRoot.Hex() {
				return fmt.Errorf("mismatch (likely manifest serialization differs)")
			}
			return nil
		})
		check("Streamed manifest is locally fetchable (DownloadData)", func() error {
			rc, err := c.File.DownloadData(ctx, streamRoot, nil)
			if err != nil {
				return err
			}
			defer rc.Close()
			b, err := io.ReadAll(rc)
			if err != nil {
				return err
			}
			if len(b) == 0 {
				return fmt.Errorf("empty body")
			}
			fmt.Printf("    manifest body=%d bytes\n", len(b))
			return nil
		})
	} else {
		skip("Stream upload", "batch not usable")
	}

	section("Encryption + redundancy round-trip (UploadData → DownloadData)")
	if batchUsable {
		payload := []byte("encrypted+redundant " + time.Now().UTC().Format(time.RFC3339Nano))
		var encRef swarm.Reference
		check("UploadData (Encrypt=true, RedundancyLevel=Medium)", func() error {
			res, err := c.File.UploadData(ctx, batchID, bytes.NewReader(payload), &api.RedundantUploadOptions{
				UploadOptions:   api.UploadOptions{Encrypt: api.BoolPtr(true)},
				RedundancyLevel: api.RedundancyLevelMedium,
			})
			if err != nil {
				return err
			}
			encRef = res.Reference
			if l := len(res.Reference.Raw()); l != swarm.EncryptedReferenceLength {
				return fmt.Errorf("expected encrypted reference (%d bytes), got %d", swarm.EncryptedReferenceLength, l)
			}
			fmt.Printf("    encryptedRef=%s (len=%d)\n", res.Reference.Hex(), len(res.Reference.Raw()))
			return nil
		})
		check("DownloadData (encrypted round-trip equality)", func() error {
			rc, err := c.File.DownloadData(ctx, encRef, nil)
			if err != nil {
				return err
			}
			defer rc.Close()
			got, err := io.ReadAll(rc)
			if err != nil {
				return err
			}
			if !bytes.Equal(got, payload) {
				return fmt.Errorf("payload mismatch (%d vs %d bytes)", len(got), len(payload))
			}
			return nil
		})
	} else {
		skip("Encryption + redundancy round-trip", "batch not usable")
	}

	section("Feed round-trip (UpdateFeed → FetchLatestFeedUpdate → IsFeedRetrievable)")
	if batchUsable {
		ecdsaKey, err := crypto.GenerateKey()
		if err != nil {
			fatalf("GenerateKey for feed: %v", err)
		}
		signer, err := swarm.NewPrivateKey(crypto.FromECDSA(ecdsaKey))
		if err != nil {
			fatalf("swarm.NewPrivateKey: %v", err)
		}
		owner := signer.PublicKey().Address()
		feedTopic := swarm.TopicFromString("integration-feed-" + time.Now().Format(time.RFC3339Nano))
		feedPayload := []byte("feed payload " + time.Now().UTC().Format(time.RFC3339Nano))

		check("UpdateFeed (idx=0)", func() error {
			res, err := c.File.UpdateFeed(ctx, batchID, signer, feedTopic, feedPayload)
			if err != nil {
				return err
			}
			fmt.Printf("    feedUpdateRef=%s\n", res.Reference.Hex())
			return nil
		})
		check("FetchLatestFeedUpdate (idx + payload equality)", func() error {
			upd, err := c.File.FetchLatestFeedUpdate(ctx, owner, feedTopic)
			if err != nil {
				return err
			}
			if upd.Index != 0 {
				return fmt.Errorf("index=%d want 0", upd.Index)
			}
			if upd.IndexNext != 1 {
				return fmt.Errorf("indexNext=%d want 1", upd.IndexNext)
			}
			if len(upd.Payload) < 8 {
				return fmt.Errorf("payload too short (%d bytes)", len(upd.Payload))
			}
			if !bytes.Equal(upd.Payload[8:], feedPayload) {
				return fmt.Errorf("payload mismatch")
			}
			fmt.Printf("    idx=%d idxNext=%d payload=%dB\n", upd.Index, upd.IndexNext, len(upd.Payload))
			return nil
		})
		check("IsFeedRetrievable (idx=0)", func() error {
			idx := uint64(0)
			ok, err := c.File.IsFeedRetrievable(ctx, owner, feedTopic, &idx, nil)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("feed not retrievable")
			}
			return nil
		})
	} else {
		skip("Feed round-trip", "batch not usable")
	}

	section("PSS send/subscribe (single-node loopback)")
	if batchUsable {
		addr, err := c.Debug.Addresses(ctx)
		if err != nil {
			fatalf("Addresses: %v", err)
		}
		// Target = first 2 bytes (4 hex chars) of own overlay so Bee delivers
		// the message back to us.
		target := addr.Overlay[:4]
		pssTopic := swarm.TopicFromString("integration-pss-" + time.Now().Format(time.RFC3339Nano))
		pssPayload := []byte("pss hello " + time.Now().UTC().Format(time.RFC3339Nano))

		check("PSS subscribe → send → receive (timeout 30s)", func() error {
			subCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			sub, err := c.PSS.PssSubscribe(subCtx, pssTopic)
			if err != nil {
				return fmt.Errorf("subscribe: %w", err)
			}
			defer sub.Cancel()

			// Brief moment for the websocket to register on the server.
			time.Sleep(300 * time.Millisecond)

			if err := c.PSS.PssSend(ctx, batchID, pssTopic, target, bytes.NewReader(pssPayload), swarm.PublicKey{}); err != nil {
				return fmt.Errorf("send: %w", err)
			}
			select {
			case msg, ok := <-sub.Messages:
				if !ok {
					return fmt.Errorf("subscription closed before message")
				}
				if !bytes.Equal(msg, pssPayload) {
					return fmt.Errorf("payload mismatch (got %d bytes, want %d)", len(msg), len(pssPayload))
				}
				fmt.Printf("    received %d bytes via target=%s\n", len(msg), target)
				return nil
			case e := <-sub.Errors:
				return fmt.Errorf("subscription error: %w", e)
			case <-time.After(30 * time.Second):
				return fmt.Errorf("timeout waiting for PSS message")
			}
		})
	} else {
		skip("PSS send/subscribe", "batch not usable")
	}

	section("GSOC send/subscribe (single-node loopback, mined signer)")
	if batchUsable {
		addr, err := c.Debug.Addresses(ctx)
		if err != nil {
			fatalf("Addresses: %v", err)
		}
		overlayBytes, err := hex.DecodeString(addr.Overlay)
		if err != nil {
			fatalf("decode overlay: %v", err)
		}
		gsocIDBytes := make([]byte, 32)
		copy(gsocIDBytes, []byte("integration-gsoc-"+time.Now().Format(time.RFC3339Nano)))
		identifier, err := swarm.NewIdentifier(gsocIDBytes)
		if err != nil {
			fatalf("NewIdentifier: %v", err)
		}
		gsocPayload := []byte("gsoc hello " + time.Now().UTC().Format(time.RFC3339Nano))

		var minedSigner swarm.PrivateKey
		check("GSOCMine (proximity 12, bee-js default)", func() error {
			ecdsaKey, err := swarm.GSOCMine(overlayBytes, gsocIDBytes, 12)
			if err != nil {
				return err
			}
			signer, err := swarm.NewPrivateKey(crypto.FromECDSA(ecdsaKey))
			if err != nil {
				return err
			}
			minedSigner = signer
			fmt.Printf("    minedOwner=%s\n", signer.PublicKey().Address().Hex())
			return nil
		})
		check("GSOC subscribe → send → receive (timeout 60s)", func() error {
			subCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			owner := minedSigner.PublicKey().Address()
			sub, err := c.GSOC.Subscribe(subCtx, owner, identifier)
			if err != nil {
				return fmt.Errorf("subscribe: %w", err)
			}
			defer sub.Cancel()

			time.Sleep(300 * time.Millisecond)

			if _, err := c.GSOC.Send(ctx, batchID, minedSigner, identifier, gsocPayload, nil); err != nil {
				return fmt.Errorf("send: %w", err)
			}
			expectedAddr, _ := gsoc.SOCAddress(identifier, owner)
			select {
			case msg, ok := <-sub.Messages:
				if !ok {
					return fmt.Errorf("subscription closed before message")
				}
				if !bytes.Equal(msg, gsocPayload) {
					return fmt.Errorf("payload mismatch (got %d bytes, want %d)", len(msg), len(gsocPayload))
				}
				fmt.Printf("    received %d bytes at socAddr=%s\n", len(msg), expectedAddr.Hex())
				return nil
			case e := <-sub.Errors:
				return fmt.Errorf("subscription error: %w", e)
			case <-time.After(60 * time.Second):
				return fmt.Errorf("timeout waiting for GSOC message")
			}
		})
	} else {
		skip("GSOC send/subscribe", "batch not usable")
	}

	section("Postage lifecycle (TopUpBatch always; DiluteBatch only on a fresh batch)")
	if batchUsable {
		check("TopUpBatch (+100 PLUR per chunk)", func() error {
			before, err := c.Postage.GetPostageBatch(ctx, batchID)
			if err != nil {
				return err
			}
			// Bee's POST /stamps/topup returns 202 Accepted once the on-chain
			// tx is queued. The /stamps amount only updates after the next
			// block (5-12s on Gnosis/Sepolia), and the test would race the
			// chain. We verify the request succeeded; the chain effect is a
			// node concern beyond this client's surface.
			if err := c.Postage.TopUpBatch(ctx, batchID, big.NewInt(100)); err != nil {
				return err
			}
			fmt.Printf("    accepted; pre-call amount=%s (chain confirmation not asserted)\n", before.Amount.String())
			return nil
		})
		if boughtBatch {
			check("DiluteBatch (depth+1)", func() error {
				before, err := c.Postage.GetPostageBatch(ctx, batchID)
				if err != nil {
					return err
				}
				newDepth := before.Depth + 1
				if err := c.Postage.DiluteBatch(ctx, batchID, newDepth); err != nil {
					return err
				}
				after, err := c.Postage.GetPostageBatch(ctx, batchID)
				if err != nil {
					return err
				}
				if after.Depth != newDepth {
					return fmt.Errorf("depth: %d → %d, want %d", before.Depth, after.Depth, newDepth)
				}
				fmt.Printf("    depth: %d → %d\n", before.Depth, after.Depth)
				return nil
			})
		} else {
			skip("DiluteBatch", "BEE_BATCH_ID is reused — declining to mutate user-supplied batch")
		}
	} else {
		skip("Postage lifecycle", "batch not usable")
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("%d passed, %d failed\n", pass, fail)
	if fail > 0 {
		os.Exit(1)
	}
}

func section(title string) {
	fmt.Println()
	fmt.Printf("== %s ==\n", title)
}

func check(name string, fn func() error) {
	err := fn()
	if err != nil {
		fmt.Printf("  ✗ %s — %v\n", name, err)
		fail++
		return
	}
	fmt.Printf("  ✓ %s\n", name)
	pass++
}

func skip(name string, reason string) {
	fmt.Printf("  - %s (skipped: %s)\n", name, reason)
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(2)
}
