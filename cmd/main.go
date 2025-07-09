package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	clavata "github.com/clavataai/gosdk"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("CLAVATA_API_KEY")
	if apiKey == "" {
		log.Fatal("CLAVATA_API_KEY environment variable is required")
	}

	policyID := os.Getenv("CLAVATA_POLICY_ID")
	if policyID == "" {
		log.Fatal("CLAVATA_POLICY_ID environment variable is required")
	}

	fmt.Printf("Testing Clavata SDK against production servers...\n")
	fmt.Printf("Policy ID: %s\n", policyID)
	fmt.Printf("API Key: %s...\n", apiKey[:min(len(apiKey), 10)])

	// Create client with default settings (production endpoint)
	client, err := clavata.New(clavata.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test content
	testContent := clavata.NewTextContent("Hello, this is a test message for content moderation.")

	fmt.Printf("\nEvaluating content: \"Hello, this is a test message for content moderation.\"\n")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Evaluate the content
	report, err := client.EvaluateOne(ctx, policyID, testContent, clavata.JobOptions{})
	if err != nil {
		// Check for precheck failures
		if errors.Is(err, clavata.ErrContentCSAM) {
			fmt.Printf("❌ Content failed precheck: CSAM detected\n")
			return
		}
		if errors.Is(err, clavata.ErrContentInvalidFormat) {
			fmt.Printf("❌ Content failed precheck: Invalid format\n")
			return
		}
		if errors.Is(err, clavata.ErrContentInvalidData) {
			fmt.Printf("❌ Content failed precheck: Invalid data\n")
			return
		}

		// Check for ContentError to get more details
		var contentErr *clavata.ContentError
		if errors.As(err, &contentErr) {
			fmt.Printf("❌ Content failed precheck: %v\n", contentErr.Err)
			fmt.Printf("   Content hash: %s\n", contentErr.ContentHash)
			return
		}

		// Other errors
		log.Fatalf("❌ Evaluation failed: %v", err)
	}

	// Success! Display results
	fmt.Printf("\n✅ Evaluation completed successfully!\n")
	fmt.Printf("Policy ID: %s\n", report.PolicyID)
	fmt.Printf("Policy Version: %s\n", report.PolicyVersionID)
	fmt.Printf("Content Hash: %s\n", report.ContentHash)
	fmt.Printf("Overall Result: %s\n", report.Result)
	fmt.Printf("Threshold: %.2f\n", report.Threshold)

	// Show label matches
	if len(report.Matches) > 0 {
		fmt.Printf("\n📊 Labels that exceeded threshold:\n")
		for labelName, score := range report.Matches {
			fmt.Printf("  • %s: %.3f\n", labelName, score)
		}
	} else {
		fmt.Printf("\n📊 No labels exceeded the threshold\n")
	}

	// Show actions
	if len(report.Actions) > 0 {
		fmt.Printf("\n🔧 Recommended actions:\n")
		for _, action := range report.Actions {
			fmt.Printf("  • %s\n", action)
		}
	}

	// Show all label reports
	fmt.Printf("\n📋 All label reports:\n")
	for labelName, labelReport := range report.LabelReports {
		fmt.Printf("  • %s: %.3f (%s) - %s\n",
			labelName,
			labelReport.Score,
			labelReport.Outcome,
			labelReport.Actions)
	}

	// Show content metadata if any
	if len(report.ContentMetadata) > 0 {
		fmt.Printf("\n🏷️  Content metadata:\n")
		for key, value := range report.ContentMetadata {
			fmt.Printf("  • %s: %s\n", key, value)
		}
	}

	fmt.Printf("\n🎉 SDK test completed successfully!\n")
}
