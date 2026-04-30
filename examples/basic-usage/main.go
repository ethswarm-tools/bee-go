package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ethswarm-tools/bee-go"
)

func main() {
	// Connect to the Bee Debug API (typically port 1635)
	// The Debug API provides health and node information
	client, err := bee.NewClient("http://localhost:1633")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// 1. Check if the Bee node is healthy
	fmt.Println("Checking node health...")
	isHealthy, err := client.Debug.Health(context.Background())
	if err != nil {
		log.Fatalf("Health check failed. Is the node running? Error: %v", err)
	}
	fmt.Printf("Node Health: %v\n\n", isHealthy)

	// 2. Fetch basic Node Information
	fmt.Println("Fetching node information...")
	info, err := client.Debug.NodeInfo(context.Background())
	if err != nil {
		log.Fatalf("Failed to get node info: %v", err)
	}

	fmt.Println("Node Information:")
	fmt.Printf("- Bee Mode: %s\n", info.BeeMode)
	if info.SwapEnabled {
		fmt.Println("- SWAP is enabled")
	} else {
		fmt.Println("- SWAP is not enabled")
	}
}
