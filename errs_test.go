package clavata

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/runtime/protoiface"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	gatewayv1 "github.com/clavataai/monorail/libs/protobufs/gateway/v1"
)

func TestLookForPrecheckFailures(t *testing.T) {
	tests := []struct {
		name           string
		inputError     error
		expectError    bool
		expectCSAM     bool
		expectInvalid  bool
		expectFormat   bool
		expectHash     string
		expectOriginal bool // Should return original error unchanged
	}{
		{
			name:           "nil error",
			inputError:     nil,
			expectError:    false,
			expectOriginal: false,
		},
		{
			name:           "non-gRPC error",
			inputError:     errors.New("regular error"),
			expectError:    true,
			expectOriginal: true,
		},
		{
			name:           "gRPC error with wrong code",
			inputError:     status.Error(codes.InvalidArgument, "wrong code"),
			expectError:    true,
			expectOriginal: true,
		},
		{
			name:           "gRPC canceled error with no details",
			inputError:     status.Error(codes.Canceled, "precheck failed"),
			expectError:    true,
			expectOriginal: true,
		},
		{
			name:        "CSAM precheck failure",
			inputError:  createPrecheckFailureError(gatewayv1.PrecheckFailureType_PRECHECK_FAILURE_TYPE_NCMEC, "hash123"),
			expectError: true,
			expectCSAM:  true,
			expectHash:  "hash123",
		},
		{
			name:         "invalid format precheck failure",
			inputError:   createPrecheckFailureError(gatewayv1.PrecheckFailureType_PRECHECK_FAILURE_TYPE_UNSUPPORTED_IMAGE_FORMAT, "hash456"),
			expectError:  true,
			expectFormat: true,
			expectHash:   "hash456",
		},
		{
			name:          "invalid data precheck failure",
			inputError:    createPrecheckFailureError(gatewayv1.PrecheckFailureType_PRECHECK_FAILURE_TYPE_INVALID_IMAGE, "hash789"),
			expectError:   true,
			expectInvalid: true,
			expectHash:    "hash789",
		},
		{
			name:        "multiple precheck failures - first one wins",
			inputError:  createMultiplePrecheckFailureError(),
			expectError: true,
			expectCSAM:  true,
			expectHash:  "first-hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lookForPrecheckFailures(tt.inputError)

			if !tt.expectError {
				assert.NoError(t, result)
				return
			}

			require.Error(t, result)

			if tt.expectOriginal {
				// Should return the original error unchanged
				assert.Equal(t, tt.inputError, result)
				return
			}

			// Should be a ContentError
			var contentErr *ContentError
			require.True(t, errors.As(result, &contentErr), "Expected ContentError, got %T", result)

			// Check the content hash
			assert.Equal(t, tt.expectHash, contentErr.ContentHash)

			// Check the specific error type
			switch {
			case tt.expectCSAM:
				assert.True(t, errors.Is(contentErr.Err, ErrContentCSAM))
				assert.True(t, errors.Is(result, ErrContentCSAM)) // Should work through Unwrap
			case tt.expectFormat:
				assert.True(t, errors.Is(contentErr.Err, ErrContentInvalidFormat))
				assert.True(t, errors.Is(result, ErrContentInvalidFormat)) // Should work through Unwrap
			case tt.expectInvalid:
				assert.True(t, errors.Is(contentErr.Err, ErrContentInvalidData))
				assert.True(t, errors.Is(result, ErrContentInvalidData)) // Should work through Unwrap
			}
		})
	}
}

func TestContentError(t *testing.T) {
	t.Run("Error method", func(t *testing.T) {
		contentErr := &ContentError{
			Err:         ErrContentCSAM,
			ContentHash: "test-hash",
		}

		assert.Contains(t, contentErr.Error(), "content error")
		assert.Contains(t, contentErr.Error(), ErrContentCSAM.Error())
	})

	t.Run("Unwrap method", func(t *testing.T) {
		contentErr := &ContentError{
			Err:         ErrContentInvalidFormat,
			ContentHash: "test-hash",
		}

		assert.Equal(t, ErrContentInvalidFormat, contentErr.Unwrap())
	})

	t.Run("errors.Is works through Unwrap", func(t *testing.T) {
		contentErr := &ContentError{
			Err:         ErrContentCSAM,
			ContentHash: "test-hash",
		}

		// Direct checking should work
		assert.True(t, errors.Is(contentErr, ErrContentCSAM))
		assert.False(t, errors.Is(contentErr, ErrContentInvalidFormat))
	})

	t.Run("errors.As extraction", func(t *testing.T) {
		originalErr := &ContentError{
			Err:         ErrContentInvalidData,
			ContentHash: "test-hash-123",
		}

		var extractedErr *ContentError
		assert.True(t, errors.As(originalErr, &extractedErr))
		assert.Equal(t, "test-hash-123", extractedErr.ContentHash)
		assert.Equal(t, ErrContentInvalidData, extractedErr.Err)
	})
}

func TestContentErrorFromPrecheckFailure(t *testing.T) {
	tests := []struct {
		name          string
		failureType   gatewayv1.PrecheckFailureType
		contentHash   string
		expectedError error
	}{
		{
			name:          "CSAM failure",
			failureType:   gatewayv1.PrecheckFailureType_PRECHECK_FAILURE_TYPE_NCMEC,
			contentHash:   "csam-hash",
			expectedError: ErrContentCSAM,
		},
		{
			name:          "Invalid image failure",
			failureType:   gatewayv1.PrecheckFailureType_PRECHECK_FAILURE_TYPE_INVALID_IMAGE,
			contentHash:   "invalid-hash",
			expectedError: ErrContentInvalidData,
		},
		{
			name:          "Unsupported format failure",
			failureType:   gatewayv1.PrecheckFailureType_PRECHECK_FAILURE_TYPE_UNSUPPORTED_IMAGE_FORMAT,
			contentHash:   "format-hash",
			expectedError: ErrContentInvalidFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			precheckFailure := &gatewayv1.PrecheckFailure{
				Type:    tt.failureType,
				Message: "precheck failed for content",
				Details: structpb.NewStringValue(tt.contentHash),
			}

			contentErr := &ContentError{}
			result := contentErr.FromPrecheckFailure(precheckFailure)

			// Should return the ContentError itself
			assert.Equal(t, contentErr, result)

			// Check the fields are set correctly
			assert.Equal(t, tt.expectedError, contentErr.Err)
			assert.Equal(t, tt.contentHash, contentErr.ContentHash)
		})
	}
}

// Helper function to create a precheck failure error as the server would
func createPrecheckFailureError(failureType gatewayv1.PrecheckFailureType, contentHash string) error {
	precheckFailure := &gatewayv1.PrecheckFailure{
		Type:    failureType,
		Message: "precheck failed for content",
		Details: structpb.NewStringValue(contentHash),
	}

	st, err := status.New(codes.Canceled, "precheck failed").WithDetails(
		protoiface.MessageV1(precheckFailure),
	)
	if err != nil {
		panic(err) // This should never happen in tests
	}

	return st.Err()
}

// Helper function to create multiple precheck failures (to test that first one wins)
func createMultiplePrecheckFailureError() error {
	firstFailure := &gatewayv1.PrecheckFailure{
		Type:    gatewayv1.PrecheckFailureType_PRECHECK_FAILURE_TYPE_NCMEC,
		Message: "precheck failed for content",
		Details: structpb.NewStringValue("first-hash"),
	}

	secondFailure := &gatewayv1.PrecheckFailure{
		Type:    gatewayv1.PrecheckFailureType_PRECHECK_FAILURE_TYPE_INVALID_IMAGE,
		Message: "precheck failed for content",
		Details: structpb.NewStringValue("second-hash"),
	}

	st, err := status.New(codes.Canceled, "precheck failed").WithDetails(
		protoiface.MessageV1(firstFailure),
		protoiface.MessageV1(secondFailure),
	)
	if err != nil {
		panic(err) // This should never happen in tests
	}

	return st.Err()
}

// Integration test to verify the complete error handling flow
func TestPrecheckFailureIntegration(t *testing.T) {
	// Simulate what the server would send
	serverError := createPrecheckFailureError(
		gatewayv1.PrecheckFailureType_PRECHECK_FAILURE_TYPE_NCMEC,
		"integration-test-hash",
	)

	// Process it through the SDK's error handling
	clientError := lookForPrecheckFailures(serverError)

	// Verify it's the expected ContentError
	require.Error(t, clientError)

	// Test direct error checking (the convenient way)
	assert.True(t, errors.Is(clientError, ErrContentCSAM))
	assert.False(t, errors.Is(clientError, ErrContentInvalidFormat))

	// Test extracting ContentError for details (the detailed way)
	var contentErr *ContentError
	require.True(t, errors.As(clientError, &contentErr))
	assert.Equal(t, "integration-test-hash", contentErr.ContentHash)
	assert.Equal(t, ErrContentCSAM, contentErr.Err)

	// Verify error message contains useful information
	assert.Contains(t, clientError.Error(), "content error")
}

// TestPrecheckFailureWireTransmission tests the scenario where precheck failures
// are transmitted over the wire and arrive as *anypb.Any (not direct protobuf types).
// This simulates real gRPC wire transmission more accurately.
func TestPrecheckFailureWireTransmission(t *testing.T) {
	tests := []struct {
		name          string
		failureType   gatewayv1.PrecheckFailureType
		contentHash   string
		expectedError error
	}{
		{
			name:          "CSAM failure over wire",
			failureType:   gatewayv1.PrecheckFailureType_PRECHECK_FAILURE_TYPE_NCMEC,
			contentHash:   "wire-csam-hash",
			expectedError: ErrContentCSAM,
		},
		{
			name:          "Invalid image failure over wire",
			failureType:   gatewayv1.PrecheckFailureType_PRECHECK_FAILURE_TYPE_INVALID_IMAGE,
			contentHash:   "wire-invalid-hash",
			expectedError: ErrContentInvalidData,
		},
		{
			name:          "Unsupported format failure over wire",
			failureType:   gatewayv1.PrecheckFailureType_PRECHECK_FAILURE_TYPE_UNSUPPORTED_IMAGE_FORMAT,
			contentHash:   "wire-format-hash",
			expectedError: ErrContentInvalidFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create error that simulates wire transmission (using *anypb.Any)
			wireError := createWireTransmissionError(tt.failureType, tt.contentHash)

			// Process it through the SDK's error handling
			clientError := lookForPrecheckFailures(wireError)

			// Verify it's the expected ContentError
			require.Error(t, clientError)

			// Test direct error checking (the convenient way)
			assert.True(t, errors.Is(clientError, tt.expectedError))

			// Test extracting ContentError for details (the detailed way)
			var contentErr *ContentError
			require.True(t, errors.As(clientError, &contentErr))
			assert.Equal(t, tt.contentHash, contentErr.ContentHash)
			assert.Equal(t, tt.expectedError, contentErr.Err)

			// Verify error message contains useful information
			assert.Contains(t, clientError.Error(), "content error")
		})
	}
}

// Helper function to create a precheck failure error as it would arrive over the wire
// (wrapped in *anypb.Any, requiring UnmarshalTo)
func createWireTransmissionError(failureType gatewayv1.PrecheckFailureType, contentHash string) error {
	precheckFailure := &gatewayv1.PrecheckFailure{
		Type:    failureType,
		Message: "precheck failed for content",
		Details: structpb.NewStringValue(contentHash),
	}

	// Marshal to Any (simulating wire transmission)
	anyDetails, err := anypb.New(precheckFailure)
	if err != nil {
		panic(err) // This should never happen in tests
	}

	st, err := status.New(codes.Canceled, "precheck failed").WithDetails(anyDetails)
	if err != nil {
		panic(err) // This should never happen in tests
	}

	return st.Err()
}

// ExampleContentError demonstrates how to handle precheck failures.
func ExampleContentError() {
	// Simulate an error that might occur during content evaluation
	err := errors.New("simulated precheck failure")

	// In practice, this would be an error returned from client.EvaluateOne()
	// or client.Evaluate()

	// Check for specific content errors
	if errors.Is(err, ErrContentCSAM) {
		fmt.Println("Content contains CSAM and cannot be evaluated")
		return
	}

	if errors.Is(err, ErrContentInvalidFormat) {
		fmt.Println("Content has an invalid format")
		return
	}

	// Or extract the ContentError for more details
	var contentErr *ContentError
	if errors.As(err, &contentErr) {
		fmt.Printf("Content %s failed precheck: %v\n", contentErr.ContentHash, contentErr.Err)
		return
	}

	// Handle other types of errors
	fmt.Printf("Other error: %v\n", err)
	// Output: Other error: simulated precheck failure
}
