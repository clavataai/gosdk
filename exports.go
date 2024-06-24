package sdk

import "github.com/clavataai/monorail/sdk/internal"

// Client is the interface for the SDK client
type Client = internal.Client

// NewClient creates a new SDK client
var NewClient = internal.NewClient
