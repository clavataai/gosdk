# Clavata Go SDK

The official Go SDK for the Clavata API.

## Installation

```bash
go get github.com/clavataai/gosdk
```

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    clavata "github.com/clavataai/gosdk"
)

func main() {
    // Create a new client
    client, err := clavata.New(
        clavata.WithAPIKey("your-api-key"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Evaluate content
    report, err := client.EvaluateOne(
        context.Background(),
        "your-policy-id",
        clavata.NewTextContent("Hello, world!"),
        clavata.JobOptions{},
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Result: %s\n", report.Result)
}
```

## Builders

To make it easier to construct complex requests, the SDK provides request **builders** for the `CreateJob`, `Evaluate` and `ListJobs` calls. Here's an example of how to use `EvaluateRequestBuilder` to simplify creation:

```go
b := NewEvaluateRequestBuilder()
b.PolicyID("my-policy-id").AddContent(
    NewTextContent("Hello world!"),
    NewImageContent(imageData),
    NewImageURLContent("https://foobar.com/path/to/image.webp")
).Expedited(true)

request := b.Build()

// Now you can use the request with:
client.Evaluate(request)
```

> Note: You can always construct request objects directly. The builders are just a different API that avoids the need to allocate slices and other complex objects.

## Retry Configuration

The SDK automatically retries requests that fail due to rate limits (HTTP 429 / gRPC RESOURCE_EXHAUSTED). By default, it uses exponential backoff with the following settings:

- Maximum retries: 5
- Initial interval: 1 second
- Maximum interval: 30 seconds
- Multiplier: 2.0
- Randomization factor: 0.1 (adds jitter to prevent thundering herd)

### Customizing Retry Behavior

You can customize the retry behavior when creating the client:

```go
client, err := clavata.New(
    clavata.WithAPIKey("your-api-key"),
    clavata.WithRetryConfig(clavata.RetryConfig{
        MaxRetries:          3,
        InitialInterval:     500 * time.Millisecond,
        MaxInterval:         10 * time.Second,
        Multiplier:          1.5,
        RandomizationFactor: 0.2,
    }),
)
```

### Disabling Retries

If you want to handle rate limiting yourself, you can disable automatic retries:

```go
client, err := clavata.New(
    clavata.WithAPIKey("your-api-key"),
    clavata.WithDisableRetry(),
)
```

### Retry Behavior Details

- **Unary calls**: The entire request is retried with exponential backoff
- **Streaming calls**: Only the initial connection is retried; once the stream is established, mid-stream errors are not retried
- **Context cancellation**: Retries respect context cancellation and will stop immediately if the context is cancelled

Example with context timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()

report, err := client.EvaluateOne(ctx, policyID, content, options)
if err != nil {
    // This could be a rate limit error if all retries were exhausted,
    // or a context deadline exceeded error if the timeout was reached
    log.Fatal(err)
}
```

## Documentation

For detailed API documentation, visit [docs.clavata.ai](https://docs.clavata.ai)
