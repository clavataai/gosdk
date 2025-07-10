package clavata

import (
	"errors"
	"fmt"

	gatewayv1 "github.com/clavataai/gosdk/internal/protobufs/gateway/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	ErrAPIKeyRequired   = errors.New("API key is required")
	ErrInvalidEndpoint  = errors.New("invalid endpoint")
	ErrClientConnection = errors.New("failed to create client for Clavata API")

	// Content errors
	ErrContentCSAM          = errors.New("content is CSAM, cannot evaluate")
	ErrContentInvalidFormat = errors.New("content is invalid format")
	ErrContentInvalidData   = errors.New("content incomplete or includes invalid data")
)

func lookForPrecheckFailures(err error) error {
	if err == nil {
		return nil
	}

	// Otherwise, try to extract the status returned by the server.
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	if st.Code() == codes.Canceled {
		details := st.Details()
		if len(details) == 0 {
			return err
		}

		d := details[0]

		// Try to cast directly to PrecheckFailure first
		if precheckFailure, ok := d.(*gatewayv1.PrecheckFailure); ok {
			return new(ContentError).FromPrecheckFailure(precheckFailure)
		}

		// Fall back to unwrapping from Any. When error details are transmitted over the wire, they
		// may be wrapped in an anypb.Any.
		if dd, ok := d.(*anypb.Any); ok {
			var precheckFailure gatewayv1.PrecheckFailure
			if err := dd.UnmarshalTo(&precheckFailure); err == nil {
				return new(ContentError).FromPrecheckFailure(&precheckFailure)
			}
		}
	}

	// If we get here, the code was not canceled, or we were unable to extract a precheck failure
	// from the details.
	return err
}

// ContentError is returned when content fails precheck validation before evaluation.
// It contains both the specific error and the ContentHash of the problematic content,
// allowing applications to identify and handle specific content issues.
//
// Common scenarios include:
//   - CSAM content detected (ErrContentCSAM)
//   - Invalid image format (ErrContentInvalidFormat)
//   - Corrupted or incomplete data (ErrContentInvalidData)
//
// Because ContentError implements Unwrap(), you can check error types directly:
//
//	// Direct error type checking (no content hash access)
//	if errors.Is(err, ErrContentCSAM) {
//		// Handle CSAM content
//	}
//
//	// Or extract ContentError for both error type and content hash
//	var contentErr *ContentError
//	if errors.As(err, &contentErr) {
//		log.Printf("Content %s failed precheck: %v", contentErr.ContentHash, contentErr.Err)
//	}
type ContentError struct {
	// Err is the specific error that occurred during precheck. Will be one of the ErrContent*
	// errors found in this package.
	Err error
	// ContentHash is the hash of the content that caused the failure
	ContentHash string
}

// Error returns a string representation of the error.
func (c *ContentError) Error() string {
	return fmt.Sprintf("content error: %v", c.Err)
}

// Unwrap returns the underlying error.
func (c *ContentError) Unwrap() error {
	return c.Err
}

// FromPrecheckFailure extracts the error and content hash from a precheck failure.
func (c *ContentError) FromPrecheckFailure(precheckFailure *gatewayv1.PrecheckFailure) error {
	switch precheckFailure.Type {
	case gatewayv1.PrecheckFailureType_PRECHECK_FAILURE_TYPE_NCMEC:
		c.Err = ErrContentCSAM
	case gatewayv1.PrecheckFailureType_PRECHECK_FAILURE_TYPE_INVALID_IMAGE:
		c.Err = ErrContentInvalidData
	case gatewayv1.PrecheckFailureType_PRECHECK_FAILURE_TYPE_UNSUPPORTED_IMAGE_FORMAT:
		c.Err = ErrContentInvalidFormat
	}

	// Extract the content hash from the precheck failure.
	c.ContentHash = precheckFailure.GetDetails().GetStringValue()
	return c
}
