package clavata

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsRetriableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "resource exhausted error",
			err:      status.Error(codes.ResourceExhausted, "rate limited"),
			expected: true,
		},
		{
			name:     "invalid argument error",
			err:      status.Error(codes.InvalidArgument, "bad request"),
			expected: false,
		},
		{
			name:     "internal error",
			err:      status.Error(codes.Internal, "internal error"),
			expected: false,
		},
		{
			name:     "non-grpc error",
			err:      context.DeadlineExceeded,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetriableError(tt.err)
			if got != tt.expected {
				t.Errorf("isRetriableError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRetryInterceptor(t *testing.T) {
	// Test that the retry interceptor retries on rate limit errors
	retryCount := 0
	maxRetries := 3

	interceptor := unaryInterceptorFactory("test-api-key", RetryConfig{
		MaxRetries:          uint64(maxRetries),
		InitialInterval:     10 * time.Millisecond,
		MaxInterval:         100 * time.Millisecond,
		Multiplier:          2.0,
		RandomizationFactor: 0.1,
	})

	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		retryCount++
		if retryCount <= maxRetries {
			return status.Error(codes.ResourceExhausted, "rate limited")
		}
		return nil
	}

	err := interceptor(context.Background(), "/test.method", nil, nil, nil, invoker)
	if err != nil {
		t.Errorf("expected success after retries, got error: %v", err)
	}

	if retryCount != maxRetries+1 {
		t.Errorf("expected %d attempts, got %d", maxRetries+1, retryCount)
	}
}

func TestRetryInterceptorMaxRetriesExceeded(t *testing.T) {
	// Test that the retry interceptor stops after max retries
	retryCount := 0
	maxRetries := 2

	interceptor := unaryInterceptorFactory("test-api-key", RetryConfig{
		MaxRetries:          uint64(maxRetries),
		InitialInterval:     10 * time.Millisecond,
		MaxInterval:         100 * time.Millisecond,
		Multiplier:          2.0,
		RandomizationFactor: 0.1,
	})

	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		retryCount++
		return status.Error(codes.ResourceExhausted, "rate limited")
	}

	err := interceptor(context.Background(), "/test.method", nil, nil, nil, invoker)
	if err == nil {
		t.Error("expected error after max retries exceeded")
	}

	if retryCount != maxRetries+1 {
		t.Errorf("expected %d attempts, got %d", maxRetries+1, retryCount)
	}

	// Verify it's still a rate limit error
	if !isRetriableError(err) {
		t.Error("expected retriable error to be returned")
	}
}

func TestRetryInterceptorNoRetry(t *testing.T) {
	// Test that non-retriable errors are not retried
	retryCount := 0

	interceptor := unaryInterceptorFactory("test-api-key", RetryConfig{
		MaxRetries:          5,
		InitialInterval:     10 * time.Millisecond,
		MaxInterval:         100 * time.Millisecond,
		Multiplier:          2.0,
		RandomizationFactor: 0.1,
	})

	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		retryCount++
		return status.Error(codes.InvalidArgument, "bad request")
	}

	err := interceptor(context.Background(), "/test.method", nil, nil, nil, invoker)
	if err == nil {
		t.Error("expected error to be returned")
	}

	if retryCount != 1 {
		t.Errorf("expected 1 attempt, got %d", retryCount)
	}
}

func TestRetryInterceptorContextCancellation(t *testing.T) {
	// Test that retries respect context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	retryCount := 0

	interceptor := unaryInterceptorFactory("test-api-key", RetryConfig{
		MaxRetries:          5,
		InitialInterval:     100 * time.Millisecond,
		MaxInterval:         1 * time.Second,
		Multiplier:          2.0,
		RandomizationFactor: 0.1,
	})

	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		retryCount++
		if retryCount == 2 {
			cancel()
		}
		return status.Error(codes.ResourceExhausted, "rate limited")
	}

	err := interceptor(ctx, "/test.method", nil, nil, nil, invoker)
	if err == nil {
		t.Error("expected error after context cancellation")
	}

	// Should have stopped early due to context cancellation
	if retryCount > 3 {
		t.Errorf("expected at most 3 attempts, got %d", retryCount)
	}
}
