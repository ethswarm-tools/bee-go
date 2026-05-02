// swarm-feed-rss is a read-only aggregator over N Swarm feeds.
//
// Configure feeds in feeds.json; the tool fetches the latest update
// from each (and optionally walks recent history). No signer
// required — feeds are public-by-default; anyone with the
// (owner, topic) pair can read.
//
// Usage:
//
//	swarm-feed-rss add  <name> <owner-eth-hex> <topic-string>
//	swarm-feed-rss list
//	swarm-feed-rss latest                       # latest from every feed
//	swarm-feed-rss walk <name> [--last N]       # last N indexes
//
// Environment:
//   - BEE_URL — base URL (default http://localhost:1633)
package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/file"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

const feedsFile = "feeds.json"

type feed struct {
	Name        string `json:"name"`
	OwnerHex    string `json:"owner_hex"`
	TopicString string `json:"topic_string"`
	TopicHex    string `json:"topic_hex"`
}

type config struct {
	Feeds []feed `json:"feeds"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	url := getenv("BEE_URL", "http://localhost:1633")
	args := os.Args[1:]
	if len(args) == 0 {
		return fmt.Errorf("usage: swarm-feed-rss <add|list|latest|walk> ...")
	}
	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	switch args[0] {
	case "add":
		return cmdAdd(args[1:])
	case "list":
		return cmdList()
	case "latest":
		return cmdLatest(client)
	case "walk":
		return cmdWalk(client, args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func cmdAdd(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: swarm-feed-rss add <name> <owner-eth-hex> <topic-string>")
	}
	name, ownerHex, topicStr := args[0], args[1], args[2]
	if _, err := swarm.EthAddressFromHex(ownerHex); err != nil {
		return fmt.Errorf("invalid owner hex: %w", err)
	}
	topic := swarm.TopicFromString(topicStr)
	cfg := load()
	for _, f := range cfg.Feeds {
		if f.Name == name {
			return fmt.Errorf("feed %s already exists", name)
		}
	}
	cfg.Feeds = append(cfg.Feeds, feed{
		Name:        name,
		OwnerHex:    strings.ToLower(ownerHex),
		TopicString: topicStr,
		TopicHex:    topic.Hex(),
	})
	if err := save(cfg); err != nil {
		return err
	}
	fmt.Printf("Added feed %s: owner=%s topic=%q\n", name, ownerHex, topicStr)
	return nil
}

func cmdList() error {
	cfg := load()
	if len(cfg.Feeds) == 0 {
		fmt.Println("(no feeds — `swarm-feed-rss add ...`)")
		return nil
	}
	fmt.Printf("%-20s  %-42s  topic\n", "name", "owner")
	for _, f := range cfg.Feeds {
		fmt.Printf("%-20s  %-42s  %q\n", f.Name, f.OwnerHex, f.TopicString)
	}
	return nil
}

func cmdLatest(client *bee.Client) error {
	cfg := load()
	if len(cfg.Feeds) == 0 {
		return fmt.Errorf("no feeds configured")
	}
	for _, f := range cfg.Feeds {
		fmt.Printf("=== %s ===\n", f.Name)
		owner, _ := swarm.EthAddressFromHex(f.OwnerHex)
		topic, _ := swarm.TopicFromHex(f.TopicHex)
		upd, err := client.File.FetchLatestFeedUpdate(context.Background(), owner, topic)
		if err != nil {
			fmt.Printf("  (no updates: %v)\n\n", err)
			continue
		}
		ts := decodeTS(upd.Payload)
		body := bodyAfterTS(upd.Payload)
		fmt.Printf("  index=%d index_next=%d ts=%d\n", upd.Index, upd.IndexNext, ts)
		printPayload("  ", body)
		fmt.Println()
	}
	return nil
}

func cmdWalk(client *bee.Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: swarm-feed-rss walk <name> [--last N]")
	}
	name := args[0]
	lastN := uint64(5)
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--last":
			i++
			if i >= len(args) {
				return fmt.Errorf("--last needs N")
			}
			n, err := strconv.ParseUint(args[i], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid N: %w", err)
			}
			lastN = n
		default:
			return fmt.Errorf("unknown flag: %s", args[i])
		}
	}
	cfg := load()
	var f *feed
	for i := range cfg.Feeds {
		if cfg.Feeds[i].Name == name {
			f = &cfg.Feeds[i]
			break
		}
	}
	if f == nil {
		return fmt.Errorf("no feed named %s", name)
	}
	owner, _ := swarm.EthAddressFromHex(f.OwnerHex)
	topic, _ := swarm.TopicFromHex(f.TopicHex)
	next, err := client.File.FindNextIndex(context.Background(), owner, topic)
	if err != nil {
		return fmt.Errorf("find_next_index: %w", err)
	}
	if next == 0 {
		fmt.Println("(empty feed)")
		return nil
	}
	last := next - 1
	from := uint64(0)
	if last+1 > lastN {
		from = last + 1 - lastN
	}
	fmt.Printf("walking %s indexes %d..=%d\n", name, from, last)
	reader := client.File.MakeSOCReader(owner)
	for i := from; i <= last; i++ {
		id, err := file.MakeFeedIdentifier(topic, i)
		if err != nil {
			return fmt.Errorf("make_feed_identifier: %w", err)
		}
		soc, err := reader.Download(context.Background(), id)
		if err != nil {
			fmt.Printf("  #%d: missing (%v)\n", i, err)
			continue
		}
		ts := decodeTS(soc.Payload)
		body := bodyAfterTS(soc.Payload)
		printPayload(fmt.Sprintf("  #%-4d ts=%d", i, ts), body)
	}
	return nil
}

func decodeTS(payload []byte) uint64 {
	if len(payload) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(payload[:8])
}

func bodyAfterTS(payload []byte) []byte {
	if len(payload) < 8 {
		return payload
	}
	return payload[8:]
}

func printPayload(prefix string, body []byte) {
	if utf8.Valid(body) {
		s := string(body)
		if len(s) <= 200 {
			fmt.Printf("%s %q\n", prefix, s)
		} else {
			fmt.Printf("%s %q…  (%d bytes)\n", prefix, s[:200], len(s))
		}
	} else {
		fmt.Printf("%s (%d bytes binary)\n", prefix, len(body))
	}
}

func load() *config {
	bytes, err := os.ReadFile(feedsFile)
	if err != nil {
		return &config{}
	}
	var c config
	if err := json.Unmarshal(bytes, &c); err != nil {
		return &config{}
	}
	return &c
}

func save(c *config) error {
	bytes, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(feedsFile, bytes, 0644)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
