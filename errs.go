package clavata

import "errors"

var (
	ErrAPIKeyRequired   = errors.New("API key is required")
	ErrInvalidEndpoint  = errors.New("invalid endpoint")
	ErrClientConnection = errors.New("failed to create client for Clavata API")
)
