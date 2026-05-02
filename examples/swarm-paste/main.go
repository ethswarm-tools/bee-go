// swarm-paste pipes stdin into a Bee node and prints a sharable
// /bzz/<ref> URL. Tiny pastebin-as-a-Swarm-app.
//
// Usage:
//
//	echo "hello" | go run ./examples/swarm-paste
//	cat report.md | go run ./examples/swarm-paste -- --ct text/markdown
//
// Environment:
//   - BEE_URL      — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID — usable postage batch (required)
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

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

	contentType := "text/plain"
	name := "paste.txt"
	encrypt := false
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ct", "--content-type":
			i++
			if i >= len(args) {
				return fmt.Errorf("--ct needs a value")
			}
			contentType = args[i]
		case "--name":
			i++
			if i >= len(args) {
				return fmt.Errorf("--name needs a value")
			}
			name = args[i]
		case "--encrypt":
			encrypt = true
		case "-h", "--help":
			fmt.Fprintln(os.Stderr,
				"usage: swarm-paste [--ct <mime>] [--name <filename>] [--encrypt]\n"+
					"reads stdin and uploads it as a file. prints the bzz URL on success.")
			return nil
		default:
			return fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	if len(body) == 0 {
		return fmt.Errorf("nothing on stdin")
	}

	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	opts := &api.FileUploadOptions{
		UploadOptions: api.UploadOptions{Encrypt: &encrypt},
		ContentType:   contentType,
	}
	result, err := client.File.UploadFile(context.Background(), batchID,
		bytes.NewReader(body), name, contentType, opts)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	trimmed := strings.TrimRight(url, "/")
	fmt.Printf("%d bytes uploaded (%s)\n", len(body), contentType)
	fmt.Printf("reference: %s\n", result.Reference.Hex())
	fmt.Printf("url:       %s/bzz/%s/\n", trimmed, result.Reference.Hex())
	if encrypt {
		fmt.Println("note: 64-byte reference contains the decryption key — keep the URL secret.")
	}
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
