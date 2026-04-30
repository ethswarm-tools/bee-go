package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/ethersphere/bee-go"
)

func main() {
	// Postage batches are purchased via the Debug API (typically port 1635)
	client, err := bee.NewClient("http://localhost:1635")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// We can accept amount and depth as arguments, or use defaults
	var amount int64 = 10000000 // default amount
	var depth uint64 = 20       // default depth

	if len(os.Args) >= 2 {
		parsedAmount, err := strconv.ParseInt(os.Args[1], 10, 64)
		if err == nil {
			amount = parsedAmount
		}
	}

	if len(os.Args) >= 3 {
		parsedDepth, err := strconv.ParseUint(os.Args[2], 10, 8)
		if err == nil {
			depth = parsedDepth
		}
	}

	// Create the label (optional, but good for tracking)
	label := "bee-go-example-batch"

	fmt.Printf("Buying a postage batch...\n")
	fmt.Printf("- Amount: %d\n", amount)
	fmt.Printf("- Depth: %d\n", depth)
	fmt.Printf("- Label: %s\n\n", label)

	// Buy the batch!
	// CreatePostageBatch(ctx, amount, depth, label)
	batchID, err := client.Postage.CreatePostageBatch(context.Background(), big.NewInt(amount), uint8(depth), label)
	if err != nil {
		log.Fatalf("Failed to buy batch: %v", err)
	}

	fmt.Printf("Successfully bought batch!\n")
	fmt.Printf("Batch ID: %s\n", batchID.Hex())

	fmt.Println("\nNote: It takes a few minutes for the chain to confirm the transaction and for the batch to become usable (discoverable by the network).")

	// You can optionally poll to check when it's ready:
	fmt.Println("Polling to see if the batch is ready... (Press Ctrl+C to cancel)")
	for {
		batch, err := client.Postage.GetPostageBatch(context.Background(), batchID)
		if err == nil && batch.Usable {
			fmt.Printf("Success! Batch %s is now usable and ready for uploads!\n", batchID.Hex())
			break
		}

		fmt.Print(".")
		time.Sleep(5 * time.Second)
	}
}
