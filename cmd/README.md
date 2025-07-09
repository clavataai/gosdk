# SDK Test Tool

This directory contains a simple command-line tool to test the Clavata Go SDK against production servers.

## Building

From the SDK root directory (`go/src/sdk`):

```bash
go build -o sdk-test ./cmd/main.go
```

## Usage

1. Set your Clavata API key as an environment variable:

   ```bash
   export CLAVATA_API_KEY="your-actual-api-key-here"
   ```

2. Edit the `policyID` variable in `main.go` to use your actual policy ID:

   ```go
   policyID := "your-actual-policy-id"
   ```

3. Run the test:
   ```bash
   ./sdk-test
   ```

## What it does

The tool will:

- Create a client using the SDK with your API key
- Connect to the production Clavata servers (default endpoint)
- Evaluate a test text message against your specified policy
- Display detailed results including:
  - Policy information
  - Content hash
  - Overall evaluation result
  - Individual label scores and outcomes
  - Recommended actions
  - Content metadata (if any)

## Error handling

The tool demonstrates proper error handling for:

- Missing API key
- Connection errors
- Precheck failures (CSAM, invalid format, invalid data)
- General evaluation errors

## Sample output

```
Testing Clavata SDK against production servers...
Policy ID: your-test-policy-id
API Key: sk-abc123...

Evaluating content: "Hello, this is a test message for content moderation."

✅ Evaluation completed successfully!
Policy ID: your-test-policy-id
Policy Version: v1.2.3
Content Hash: abc123def456
Overall Result: FALSE
Threshold: 0.70

📊 No labels exceeded the threshold

📋 All label reports:
  • inappropriate: 0.123 (FALSE) - block
  • spam: 0.045 (FALSE) - flag
  • toxic: 0.089 (FALSE) - warn

🎉 SDK test completed successfully!
```

This tool is useful for:

- Verifying SDK connectivity to production servers
- Testing policy configurations
- Debugging evaluation issues
- Demonstrating SDK usage patterns
