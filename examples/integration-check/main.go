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
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	bee "github.com/ethersphere/bee-go"
	"github.com/ethersphere/bee-go/pkg/file"
	"github.com/ethersphere/bee-go/pkg/swarm"
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
