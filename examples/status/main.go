package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ethersphere/bee-go"
)

func main() {
	// The Status endpoint is part of the Debug API, typically on port 1635
	client, err := bee.NewClient("http://localhost:1633")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Println("Checking node status...")

	// Status checks the status of the Bee node components
	status, err := client.Debug.Status(context.Background())
	if err != nil {
		log.Fatalf("Failed to retrieve status: %v", err)
	}

	fmt.Printf("Node Status Output:\n")
	fmt.Printf("- Overlay: %s\n", status.Overlay)
	fmt.Printf("- Bee Mode: %s\n", status.BeeMode)
	fmt.Printf("- Connected Peers: %d\n", status.ConnectedPeers)
	fmt.Printf("- Reserve Size: %d\n", status.ReserveSize)
	fmt.Printf("- Pullsync Rate: %f\n", status.PullsyncRate)
	fmt.Printf("- Is Reachable: %v\n", status.IsReachable)

	// Readiness checks if the Bee node is ready to serve requests
	fmt.Println("\nChecking node readiness...")
	isReady, err := client.Debug.Readiness(context.Background())
	if err != nil {
		log.Fatalf("Failed to retrieve readiness: %v", err)
	}

	if isReady {
		fmt.Println("Node Readiness: Ready")
	} else {
		fmt.Println("Node Readiness: Not Ready")
	}
}
