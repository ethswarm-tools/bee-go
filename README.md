# bee-go

> A Go client library for connecting to Swarm Bee nodes.

**bee-go** provides a convenient, type-safe interface for interacting with the Bee API. It is designed to be familiar to users of `bee-js` while adhering to idiomatic Go patterns.

## Installation

```bash
go get github.com/ethersphere/bee-go
```

## Usage

### Connect to Bee

```go
package main

import (
	"fmt"
	"log"

	"github.com/ethersphere/bee-go"
)

func main() {
	// Create a new client connecting to a local Bee node
	client, err := bee.NewClient("http://localhost:1633")
	if err != nil {
		log.Fatal(err)
	}

	// Check if connected
	health, err := client.Debug.Health(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Node Health: %v\n", health)
}
```

### Postage Stamps

Before uploading data, you need a postage batch.

```go
import (
	"context"
	"math/big"
)

// Buy a new batch: amount=1000, depth=20, label="my-batch"
batchID, err := client.Postage.CreatePostageBatch(context.Background(), big.NewInt(1000), 20, "my-batch")
if err != nil {
	log.Fatal(err)
}
fmt.Printf("Batch created: %s\n", batchID)
```

### Upload Data

Upload raw data (bytes) to Swarm.

```go
import "strings"

data := strings.NewReader("Hello Swarm!")
// UploadData(ctx, batchID, dataReader, options)
ref, err := client.File.UploadData(context.Background(), batchID, data, nil)
if err != nil {
	log.Fatal(err)
}
fmt.Printf("Uploaded Reference: %s\n", ref.Value)
```

### Upload File

Upload a file with Content-Type.

```go
file, _ := os.Open("image.png")
defer file.Close()

// UploadFile(ctx, batchID, reader, filename, contentType, options)
ref, err := client.File.UploadFile(context.Background(), batchID, file, "image.png", "image/png", nil)
if err != nil {
	log.Fatal(err)
}
```

### Download Data

```go
reader, err := client.File.DownloadData(context.Background(), ref)
if err != nil {
	log.Fatal(err)
}
defer reader.Close()

data, _ := io.ReadAll(reader)
fmt.Println(string(data))
```

### Feeds

Update a feed using a private key.

```go
import (
	"github.com/ethereum/go-ethereum/crypto"
)

// Generate (or load) a private key
privKey, _ := crypto.GenerateKey()
topic := "0000000000000000000000000000000000000000000000000000000000000000" // 32-byte hex topic

// Update the feed
ref, err := client.File.UpdateFeedWithIndex(context.Background(), batchID, privKey, topic, 0, []byte("feed update data"))
if err != nil {
	log.Fatal(err)
}
fmt.Printf("Feed Updated: %s\n", ref.Value)
```

### PSS (Postal Service for Swarm)

Send a PSS message.

```go
// PssSend(ctx, topic, target, dataReader, recipient)
err := client.PSS.PssSend(context.Background(), "topic-hex", "target-prefix", strings.NewReader("message"), "recipient-key")
if err != nil {
	log.Fatal(err)
}
```

## Structure

The library is organized into packages reflecting the Bee API domains:

- **`pkg/api`**: Core API types and helpers.
- **`pkg/debug`**: Debug API endpoints (Node Info, Balance, etc.).
- **`pkg/file`**: File and data upload/download operations.
- **`pkg/postage`**: Postage batch management.
- **`pkg/swarm`**: Core Swarm primitives (BMT, SOC).
- **`pkg/pss`**: PSS messaging.

## Contribute

Contributions are welcome! Please fork the repository and submit a pull request.

## License

MIT
