// Package clavata provides the official Go SDK for the Clavata Content Moderation API.
//
// Clavata is a content moderation platform that helps you automatically detect and filter
// unwanted content in text, images, and other media. This SDK provides a simple and
// idiomatic Go interface for integrating Clavata's moderation capabilities into your applications.
//
// # Quick Start
//
// To get started, you'll need a Clavata API key. Contact support@clavata.ai to request one.
//
//	import "github.com/clavataai/gosdk"
//
//	// Create a client
//	client, err := clavata.New(clavata.WithAPIKey("your-api-key"))
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Evaluate content
//	report, err := client.EvaluateOne(
//		context.Background(),
//		"your-policy-id",
//		clavata.NewTextContent("Content to moderate"),
//		clavata.JobOptions{},
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if report.Result == clavata.OutcomeTrue {
//		fmt.Println("Content flagged:", report.Actions)
//	}
//
// # Content Types
//
// The SDK supports multiple content types:
//
//   - Text content: clavata.NewTextContent("your text")
//   - Image data: clavata.NewImageContent(imageBytes)
//   - Image files: clavata.NewImageFileContent("path/to/image.jpg")
//   - Image URLs: clavata.NewImageURLContent("https://example.com/image.jpg")
//
// # Evaluation Methods
//
// The SDK provides several ways to evaluate content:
//
//   - EvaluateOne: Evaluate a single piece of content synchronously
//   - Evaluate: Evaluate multiple pieces of content with streaming results
//   - EvaluateIter: Same as Evaluate but returns a Go iterator for easier handling
//   - CreateJob: Create an asynchronous job for batch processing
//
// # Error Handling and Retries
//
// The SDK automatically handles rate limiting and transient errors with configurable
// exponential backoff. You can customize retry behavior:
//
//	client, err := clavata.New(
//		clavata.WithAPIKey("your-api-key"),
//		clavata.WithRetryConfig(clavata.RetryConfig{
//			MaxRetries:      3,
//			InitialInterval: 500 * time.Millisecond,
//			MaxInterval:     30 * time.Second,
//			Multiplier:      2.0,
//		}),
//	)
//
// # Content Checks and Possible Failure Modes
//
// Before content evaluation, Clavata runs prechecks to identify known issues that would
// prevent evaluation. Common precheck failures include:
//
//   - CSAM content detected (matches known illegal material)
//   - Invalid image format or corrupted data
//   - Unsupported media types
//
// When a precheck failure occurs, the SDK returns a ContentError that includes both the
// specific error and the ContentHash of the problematic content:
//
//	report, err := client.EvaluateOne(ctx, policyID, content, options)
//	if err != nil {
//		// Option 1: Check specific error types directly (if you don't need content hash)
//		if errors.Is(err, clavata.ErrContentCSAM) {
//			// Handle CSAM content directly
//			return
//		}
//
//		// Option 2: Extract ContentError to get both error type and content hash
//		var contentErr *clavata.ContentError
//		if errors.As(err, &contentErr) {
//			fmt.Printf("Content failed precheck: %v\n", contentErr.Err)
//			fmt.Printf("Content hash: %s\n", contentErr.ContentHash)
//
//			// Check specific error types
//			if errors.Is(contentErr.Err, clavata.ErrContentCSAM) {
//				// Handle CSAM content
//			}
//		}
//	}
//
// # Timeouts
//
// Configure request timeouts as needed:
//
//	client, err := clavata.New(
//		clavata.WithAPIKey("your-api-key"),
//		clavata.WithTimeout(60 * time.Second),
//	)
//
// For more information and examples, visit: https://docs.clavata.ai
package clavata
