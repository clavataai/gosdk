package clavata_test

import (
	"context"
	"fmt"
	"log"

	clavata "github.com/clavataai/gosdk"
)

// Example demonstrates how to create a Clavata client and evaluate content.
func Example() {
	// Create a new client with your API key
	client, err := clavata.New(clavata.WithAPIKey("your-api-key-here"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Create some content to evaluate
	content := clavata.NewTextContent("Hello, world!")

	// Evaluate the content against a policy
	ctx := context.Background()
	report, err := client.EvaluateOne(ctx, "your-policy-id", content, clavata.JobOptions{})
	if err != nil {
		log.Printf("Error evaluating content: %v", err)
		return
	}

	// Process the report
	fmt.Printf("Policy: %s, Overall result: %s\n", report.PolicyID, report.Result)
	for labelName, score := range report.Matches {
		fmt.Printf("  %s: %.2f\n", labelName, score)
	}
}

// ExampleClient_EvaluateOne demonstrates how to evaluate a single piece of content.
func ExampleClient_EvaluateOne() {
	client, err := clavata.New(clavata.WithAPIKey("your-api-key-here"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	content := clavata.NewTextContent("Sample text to evaluate")

	report, err := client.EvaluateOne(ctx, "content-safety-policy", content, clavata.JobOptions{
		Threshold: 0.7,
		Expedited: true,
	})
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Evaluation completed for policy %s\n", report.PolicyID)
	fmt.Printf("Content hash: %s\n", report.ContentHash)
	fmt.Printf("Overall result: %s\n", report.Result)
}
