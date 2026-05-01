package bee_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

// Construct a Client against a local Bee node and check that the node
// is healthy.
func ExampleNewClient() {
	c, err := bee.NewClient("http://localhost:1633")
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	healthy, err := c.Debug.Health(ctx)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("healthy:", healthy)
}

// Upload a few bytes against an existing postage batch and print the
// returned reference.
func ExampleClient_uploadData() {
	c, err := bee.NewClient("http://localhost:1633")
	if err != nil {
		log.Fatal(err)
	}

	batchID, err := swarm.BatchIDFromHex("0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		log.Fatal(err)
	}

	res, err := c.File.UploadData(context.Background(), batchID, strings.NewReader("Hello Swarm!"), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("reference:", res.Reference.Hex())
}

// Round-trip a payload through Bee: upload bytes then download them
// back by reference.
func ExampleClient_downloadData() {
	c, err := bee.NewClient("http://localhost:1633")
	if err != nil {
		log.Fatal(err)
	}

	ref, err := swarm.ReferenceFromHex("0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		log.Fatal(err)
	}

	body, err := c.File.DownloadData(context.Background(), ref, nil)
	if err != nil {
		log.Fatal(err)
	}

	data, err := io.ReadAll(body)
	_ = body.Close()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("downloaded %d bytes\n", len(data))
}

// Buy a postage batch sized for 1 GB and lasting 30 days using current
// chain pricing. The returned BatchID is what every upload method
// expects as its stamp argument.
func ExampleClient_BuyStorage() {
	c, err := bee.NewClient("http://localhost:1633")
	if err != nil {
		log.Fatal(err)
	}

	size, err := swarm.SizeFromGigabytes(1)
	if err != nil {
		log.Fatal(err)
	}

	batchID, err := c.BuyStorage(context.Background(), size, swarm.DurationFromDays(30), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("batch:", batchID.Hex())
}

// Translate a non-2xx error from Bee into something printable. Every
// endpoint returns either nil or a *swarm.BeeError /
// *swarm.BeeArgumentError / *swarm.BeeResponseError; use errors.As to
// inspect the Bee-side context.
func ExampleClient_errors() {
	c, err := bee.NewClient("http://localhost:1633")
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.Postage.GetPostageBatches(context.Background())
	if err == nil {
		return
	}

	var rerr *swarm.BeeResponseError
	if errors.As(err, &rerr) {
		fmt.Printf("bee returned %d %s for %s %s\n",
			rerr.Status, rerr.StatusText, rerr.Method, rerr.URL)
		return
	}
	fmt.Println("other error:", err)
}
