.PHONY: build test clean help

# Default target
all: build

# Build the test tool
build:
	cd .. && go build -o sdk-test ./cmd/main.go

# Run the built binary for this tool (requires CLAVATA_API_KEY and CLAVATA_POLICY_ID environment variables)
test: build
	cd .. && ./sdk-test

# Run the test tool (requires CLAVATA_API_KEY and CLAVATA_POLICY_ID environment variables)
test-local:
	cd .. && go run ./cmd/main.go

# Clean up build artifacts
clean:
	cd .. && rm -f sdk-test

# Show help
help:
	@echo "Available targets:"
	@echo "  build  - Build the SDK test tool"
	@echo "  test   - Build and run the test tool (requires CLAVATA_API_KEY and CLAVATA_POLICY_ID)"
	@echo "  clean  - Remove build artifacts"
	@echo "  help   - Show this help message"
	@echo ""
	@echo "Usage:"
	@echo "  make build"
	@echo "  CLAVATA_API_KEY=your-key CLAVATA_POLICY_ID=your-policy-id make test"