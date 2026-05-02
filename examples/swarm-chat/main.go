// swarm-chat is a terminal chat over PSS.
//
// Each message is a tiny JSON envelope {user, ts, text} sent to a
// topic prefix. The chat client subscribes to the same topic and
// prints incoming messages while reading stdin lines to send.
//
// Bee does NOT deliver a node's own PSS messages back to itself, so
// a single process won't see what it sent. Run two instances against
// different Bee URLs (or different BEE_BATCH_IDs) and they'll talk.
//
// Usage:
//
//	swarm-chat [--user <name>] [--topic <name>] [--target <hex-prefix>]
//
// Environment:
//   - BEE_URL      — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID — usable postage batch (required)
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
	"unicode/utf8"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

type envelope struct {
	User string `json:"user"`
	TS   int64  `json:"ts"`
	Text string `json:"text"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	url := getenv("BEE_URL", "http://localhost:1633")
	batchHex := os.Getenv("BEE_BATCH_ID")
	if batchHex == "" {
		return fmt.Errorf("BEE_BATCH_ID is required")
	}
	batchID, err := swarm.BatchIDFromHex(batchHex)
	if err != nil {
		return fmt.Errorf("invalid BEE_BATCH_ID: %w", err)
	}

	user := getenv("USER", "anon")
	topicName := "swarm-chat-default"
	target := ""
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--user":
			i++
			if i >= len(args) {
				return fmt.Errorf("--user needs a value")
			}
			user = args[i]
		case "--topic":
			i++
			if i >= len(args) {
				return fmt.Errorf("--topic needs a value")
			}
			topicName = args[i]
		case "--target":
			i++
			if i >= len(args) {
				return fmt.Errorf("--target needs a value")
			}
			target = args[i]
		case "-h", "--help":
			fmt.Println("swarm-chat [--user <name>] [--topic <name>] [--target <hex-prefix>]")
			return nil
		default:
			return fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	topic := swarm.TopicFromString(topicName)
	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}

	fmt.Printf("Joining %q as %q\n", topicName, user)
	fmt.Printf("(target prefix: %q)\n", target)
	fmt.Println("Type a message and press enter to send. Ctrl+D to quit.")
	fmt.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub, err := client.PSS.PssSubscribe(ctx, topic)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}
	defer sub.Cancel()

	// Receive loop.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-sub.Messages:
				if !ok {
					return
				}
				var env envelope
				if err := json.Unmarshal(msg, &env); err == nil {
					if env.User != user {
						fmt.Printf("[%s] %s\n", env.User, env.Text)
					}
					continue
				}
				if utf8.Valid(msg) {
					fmt.Printf("[?] %s\n", string(msg))
				} else {
					fmt.Printf("[?] (%d bytes binary)\n", len(msg))
				}
			case err, ok := <-sub.Errors:
				if !ok {
					return
				}
				fmt.Fprintf(os.Stderr, "subscription error: %v\n", err)
				return
			}
		}
	}()

	// Send loop.
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		env := envelope{User: user, TS: time.Now().Unix(), Text: line}
		body, err := json.Marshal(env)
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
			continue
		}
		if err := client.PSS.PssSend(context.Background(), batchID, topic, target,
			bytes.NewReader(body), swarm.PublicKey{}); err != nil {
			fmt.Fprintf(os.Stderr, "send failed: %v\n", err)
		}
	}
	fmt.Println("\n(stdin closed, exiting.)")
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
