# Clavata Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/clavataai/gosdk.svg)](https://pkg.go.dev/github.com/clavataai/gosdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/clavataai/gosdk)](https://goreportcard.com/report/github.com/clavataai/gosdk)

The official Go SDK for the [Clavata Content Evaluation API](https://clavata.ai).

Clavata helps you automatically detect and filter unwanted content in text, images, and other media. This SDK provides a simple and idiomatic Go interface for integrating Clavata's moderation capabilities into your applications.

## Features

- ✅ **Simple API** - Easy-to-use client with sensible defaults
- ✅ **Multiple content types** - Text, images, files, and URLs
- ✅ **Flexible evaluation** - Sync, async, streaming, and batch processing
- ✅ **Robust error handling** - Automatic retries with exponential backoff
- ✅ **Type safety** - Full Go type definitions for all API responses
- ✅ **Context support** - Built-in context.Context support for timeouts and cancellation

## Installation

```bash
go get github.com/clavataai/gosdk
```

## Quick Start

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
        clavata.NewTextContent("Content to evaluate"),
        clavata.JobOptions{},
    )
    if err != nil {
        log.Fatal(err)
    }

    // Check results
    if report.Result == clavata.OutcomeTrue {
        fmt.Printf("Content flagged! Actions: %v\n", report.Actions)
        fmt.Printf("Matches: %v\n", report.Matches)
    } else {
        fmt.Println("Input did not match any labels at the current threshold of %d", report.Threshold)
    }
}
```

## Content Types

The SDK supports multiple content types:

```go
// Text content
textContent := clavata.NewTextContent("Your text here")

// Image from bytes
imageContent := clavata.NewImageContent(imageBytes)

// Image from file
fileContent := clavata.NewImageFileContent("path/to/image.jpg")

// Image from URL
urlContent := clavata.NewImageURLContent("https://example.com/image.jpg")
```

## Evaluation Methods

### Synchronous - Single Content

Perfect for real-time moderation:

```go
report, err := client.EvaluateOne(
    ctx,
    "policy-id",
    clavata.NewTextContent("content"),
    clavata.JobOptions{Threshold: 0.8},
)
```

### Streaming - Multiple Content

Ideal for processing multiple items with real-time results:

```go
request := clavata.NewEvaluateRequestBuilder().
    PolicyID("policy-id").
    AddContent(
        clavata.NewTextContent("First content"),
        clavata.NewTextContent("Second content"),
    ).
    Build()

// Using iterator (Go 1.23+)
for report, err := range client.EvaluateIter(ctx, request) {
    if err != nil {
        log.Printf("Error: %v", err)
        continue
    }
    fmt.Printf("Result: %s\n", report.Result)
}

// Using channel
ch, err := client.Evaluate(ctx, request)
if err != nil {
    log.Fatal(err)
}

for result := range ch {
    if result.Error != nil {
        log.Printf("Error: %v", result.Error)
        continue
    }
    fmt.Printf("Result: %s\n", result.Report.Result)
}
```

### Asynchronous Jobs

For batch processing large amounts of content:

```go
// Create job
job, err := client.CreateJob(ctx, &clavata.CreateJobRequest{
    PolicyID: "policy-id",
    Content: []clavata.Contenter{
        clavata.NewTextContent("content 1"),
        clavata.NewTextContent("content 2"),
    },
    WebhookURL: "https://your-app.com/webhook", // Optional
})

// Check job status
job, err = client.GetJob(ctx, job.ID)
if job.Status == clavata.JobStatusCompleted {
    for _, report := range job.Results {
        fmt.Printf("Result: %s\n", report.Result)
    }
}
```

## Configuration

### Custom Timeouts

```go
client, err := clavata.New(
    clavata.WithAPIKey("your-api-key"),
    clavata.WithTimeout(60 * time.Second),
)
```

### Custom Retry Behavior

```go
client, err := clavata.New(
    clavata.WithAPIKey("your-api-key"),
    clavata.WithRetryConfig(clavata.RetryConfig{
        MaxRetries:              5,
        InitialInterval:         500 * time.Millisecond,
        MaxInterval:            30 * time.Second,
        Multiplier:             2.0,
        RandomizationFactor:    0.1,
    }),
)
```

### Disable Retries

```go
client, err := clavata.New(
    clavata.WithAPIKey("your-api-key"),
    clavata.WithDisableRetry(),
)
```

## Error Handling

The SDK automatically handles rate limiting and transient errors with exponential backoff. For permanent errors, you'll receive detailed error information.

### Standard Error Handling

```go
report, err := client.EvaluateOne(ctx, "policy-id", content, options)
if err != nil {
    // Check for specific precheck failures directly
    if errors.Is(err, clavata.ErrContentCSAM) {
        log.Printf("CSAM content detected")
        return
    }

    // Check for other precheck failures
    if errors.Is(err, clavata.ErrContentInvalidFormat) {
        log.Printf("Invalid image format")
        return
    }

    // Or extract ContentError for both error type and content hash
    var contentErr *clavata.ContentError
    if errors.As(err, &contentErr) {
        log.Printf("Content %s failed precheck: %v", contentErr.ContentHash, contentErr.Err)
        return
    }

    // Handle other errors
    if errors.Is(err, clavata.ErrAPIKeyRequired) {
        log.Fatal("API key is required")
    }
    log.Printf("Evaluation failed: %v", err)
    return
}
```

## Precheck Failures

Before content evaluation, Clavata runs prechecks to identify known issues that would prevent evaluation. This helps catch problems early and protects against processing illegal or invalid content.

### Common Precheck Failures

- **CSAM Detection**: Content matches known Child Sexual Abuse Material
- **Invalid Image Format**: Corrupted or unsupported image data
- **Invalid Data**: Incomplete or malformed content

### Handling Precheck Failures

When a precheck failure occurs, the SDK returns a `ContentError` that includes:

- The specific error type
- The content hash of the problematic content

You can handle precheck failures in two ways:

#### Option 1: Direct Error Type Checking

If you only need to check the error type and don't need the content hash, you can check directly:

```go
report, err := client.EvaluateOne(ctx, "policy-id", content, options)
if err != nil {
    // Check specific error types directly
    switch {
    case errors.Is(err, clavata.ErrContentCSAM):
        // Handle CSAM content - do not process further
        log.Printf("CSAM content detected")
        return
    case errors.Is(err, clavata.ErrContentInvalidFormat):
        // Handle invalid format - possibly retry with different content
        log.Printf("Invalid image format")
        return
    case errors.Is(err, clavata.ErrContentInvalidData):
        // Handle invalid data - content may be corrupted
        log.Printf("Invalid content data")
        return
    }

    // Handle other errors
    log.Printf("Evaluation failed: %v", err)
    return
}
```

#### Option 2: Extract ContentError for Additional Details

If you need both the error type and content hash:

```go
report, err := client.EvaluateOne(ctx, "policy-id", content, options)
if err != nil {
    var contentErr *clavata.ContentError
    if errors.As(err, &contentErr) {
        // Handle precheck failure with access to content hash
        fmt.Printf("Content failed precheck: %v\n", contentErr.Err)
        fmt.Printf("Content hash: %s\n", contentErr.ContentHash)

        // Check specific error types
        switch {
        case errors.Is(contentErr.Err, clavata.ErrContentCSAM):
            // Handle CSAM content - do not process further
            log.Printf("CSAM content detected: %s", contentErr.ContentHash)
        case errors.Is(contentErr.Err, clavata.ErrContentInvalidFormat):
            // Handle invalid format - possibly retry with different content
            log.Printf("Invalid image format: %s", contentErr.ContentHash)
        case errors.Is(contentErr.Err, clavata.ErrContentInvalidData):
            // Handle invalid data - content may be corrupted
            log.Printf("Invalid content data: %s", contentErr.ContentHash)
        }
        return
    }

    // Handle other errors
    log.Printf("Evaluation failed: %v", err)
    return
}
```

### Error Types

The SDK defines specific error types for different precheck failures:

```go
// Content errors
var (
    ErrContentCSAM          = errors.New("content is CSAM, cannot evaluate")
    ErrContentInvalidFormat = errors.New("content is invalid format")
    ErrContentInvalidData   = errors.New("content incomplete or includes invalid data")
)
```

## Response Structure

```go
type Report struct {
    PolicyID        string            // ID of the evaluated policy
    ContentHash     string            // Hash of the content
    Result          Outcome           // Overall result (TRUE/FALSE/FAILED)
    Threshold       float64           // Threshold used for evaluation
    Matches         map[string]float64 // Labels that exceeded threshold
    Actions         []string          // Recommended actions
    LabelReports    map[string]LabelReport // Detailed per-label results
}

type LabelReport struct {
    LabelName string   // Name of the label
    Score     float64  // Score (0.0 to 1.0)
    Outcome   Outcome  // Result for this label
    Actions   string   // Actions for this label
}
```

## Requirements

- Go 1.23.0 or later
- Valid Clavata API key

## Getting an API Key

1. Sign up at [clavata.ai](https://www.clavata.ai)
2. Create a policy for your content moderation needs
3. Generate an API key in your dashboard
4. Use the policy ID and API key in your integration

## Documentation

- **API Reference**: [pkg.go.dev/github.com/clavataai/gosdk](https://pkg.go.dev/github.com/clavataai/gosdk)
- **Full Documentation**: [clavata.helpscoutdocs.com](https://clavata.helpscoutdocs.com)

## Support

- **Email**: support@clavata.ai
- **Documentation**: [docs.clavata.ai](https://docs.clavata.ai)
- **Issues**: [GitHub Issues](https://github.com/clavataai/gosdk/issues)

## License

This SDK is distributed under the Apache 2.0 License. See [LICENSE](./LICENSE-2.0.txt) for more information.

---

Made with ❤️ by the [Clavata](https://clavata.ai) team.
