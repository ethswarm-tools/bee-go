// Listen for PSS messages on a topic, or send one.
//
// Usage:
//
//	# listen
//	go run ./examples/pss-send-receive listen <topic>
//
//	# send (separate process / different Bee node)
//	go run ./examples/pss-send-receive send <topic> <target-prefix> <message>
//
// <topic> is a UTF-8 string; it is hashed via keccak256 to a 32-byte
// topic identifier (matching swarm.TopicFromString semantics).
//
// <target-prefix> is a short hex string Bee uses as a routing prefix
// (e.g. "0001"). PSS does not require knowing the recipient's full
// overlay; any node whose overlay starts with the prefix delivers
// the message.
//
// Note: a Bee node does NOT receive its own PSS messages — for a real
// demo, run the listener on one node and the sender on another. Two
// nodes pointed at different BEE_URLs, same topic.
//
// Environment:
//   - BEE_URL      — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID — required for `send` (any usable batch hex id)
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"unicode/utf8"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: pss-send-receive <listen|send> ...")
	}
	mode := os.Args[1]
	url := getenv("BEE_URL", "http://localhost:1633")

	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	switch mode {
	case "listen":
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: pss-send-receive listen <topic>")
		}
		topicStr := os.Args[2]
		topic := swarm.TopicFromString(topicStr)
		fmt.Printf("Subscribing to topic %q (%s) on %s...\n", topicStr, topic.Hex(), url)
		fmt.Println("Press Ctrl+C to stop.")
		fmt.Println()

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()

		sub, err := client.PSS.PssSubscribe(ctx, topic)
		if err != nil {
			return fmt.Errorf("subscribe failed: %w", err)
		}
		defer sub.Cancel()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("Subscription cancelled.")
				return nil
			case msg, ok := <-sub.Messages:
				if !ok {
					fmt.Println("Subscription closed.")
					return nil
				}
				if utf8.Valid(msg) {
					fmt.Printf("[%d bytes] %q\n", len(msg), string(msg))
				} else {
					fmt.Printf("[%d bytes] (binary)\n", len(msg))
				}
			case err, ok := <-sub.Errors:
				if !ok {
					return nil
				}
				return fmt.Errorf("subscription error: %w", err)
			}
		}

	case "send":
		if len(os.Args) < 5 {
			return fmt.Errorf("usage: pss-send-receive send <topic> <target-prefix> <message>")
		}
		topicStr := os.Args[2]
		target := os.Args[3]
		message := os.Args[4]

		batchHex := os.Getenv("BEE_BATCH_ID")
		if batchHex == "" {
			return fmt.Errorf("BEE_BATCH_ID is required for send (set to a usable batch hex id)")
		}
		batchID, err := swarm.BatchIDFromHex(batchHex)
		if err != nil {
			return fmt.Errorf("invalid BEE_BATCH_ID: %w", err)
		}
		topic := swarm.TopicFromString(topicStr)

		fmt.Println("Sending PSS message")
		fmt.Printf("- URL:    %s\n", url)
		fmt.Printf("- Topic:  %s (from %q)\n", topic.Hex(), topicStr)
		fmt.Printf("- Target: %s\n", target)
		fmt.Printf("- Batch:  %s\n", batchID.Hex())
		fmt.Printf("- Body:   %q\n\n", message)

		body := bytes.NewReader([]byte(message))
		if err := client.PSS.PssSend(context.Background(), batchID, topic, target, body, swarm.PublicKey{}); err != nil {
			return fmt.Errorf("send failed: %w", err)
		}
		fmt.Println("Message sent.")
		return nil

	default:
		return fmt.Errorf("unknown mode %q (expected listen or send)", mode)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
