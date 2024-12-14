package internal

import "context"

type Client interface {
	// Close closes the client connection
	Close() error

	// Evaluate evaluates one or more pieces of content against one or more policies
	Evaluate(ctx context.Context, inputs *EvaluateInput) (*EvaluateOutput, error)
}

// ContentMode is the type of content being evaluated
type ContentMode uint8

const (
	// Unspecified indicates not set
	ContentModeUnspecified ContentMode = iota
	// Plain text
	ContentModeText
	// Image data encoded as bytes
	ContentModeImage
	// Image data as a URL
	ContentModeImageUrl
	// Image data as a base64 encoded string
	ContentModeImageBase64
)

func (m ContentMode) String() string {
	return [...]string{"text", "image", "image_url", "image_base64"}[m]
}

func (m *ContentMode) FromString(s string) ContentMode {
	switch s {
	case "text":
		*m = ContentModeText
	case "image":
		*m = ContentModeImage
	case "image_url":
		*m = ContentModeImageUrl
	case "image_base64":
		*m = ContentModeImageBase64
	}
	return *m
}

func (m ContentMode) MarshalJSON() ([]byte, error) {
	return []byte(`"` + m.String() + `"`), nil
}

type (
	// PolicyName is the name of a policy, its an alias that is used for clarity
	PolicyName = string
	// SectionName is the name of a section, its an alias that is used for clarity
	SectionName = string
)

type SectionOutcome struct {
	// SectionName is the name of the section
	SectionName SectionName

	// Result is the result of the evaluation
	Result bool

	// Message is the user provided value associated with the Secction
	Message string
}

// Outcome is the result of evaluating content against a policy. Since multiple pieces of content
// can be provided in a single request as well as multiple policies, the outcome contains both
// the content hash and the policy name. This allows the caller to determine which content and
// policy the outcome is associated with.
type Outcome struct {
	// ContentHash is the hash of the content
	ContentHash string

	// PolicyName is the name of the policy
	PolicyName PolicyName

	// Result is the result of the evaluation
	Result bool

	// Sections provides the outcomes, by section, for the policy
	Sections map[SectionName]*SectionOutcome
}

type Content struct {
	// Mode defines the type of the content (i.e., text, image, image_url, etc.)
	Mode ContentMode

	// Payload is the content to evaluate. Regardless of the type, the content is encoded into
	// a byte array as all content modes can be represented as bytes. The server will
	// decode the content based on the mode.
	Payload []byte
}

// EvaluateInput is the input to the Evaluate method
type EvaluateInput struct {
	// Content is the content to evaluate
	Content []Content

	// PolicyId is the ID of the policy to use
	PolicyId string
}

type EvaluateOutput struct {
	// JobUuid is the unique identifier for the job
	JobUuid string

	// Outcomes is a list of outcomes, by policy, for the content
	Outcomes []*Outcome
}
