package askai

import (
	"context"
	"errors"
)

// Sentinel errors
var (
	// ErrAskAI is the base sentinel for all askai package errors
	ErrAskAI = errors.New("error in Ask AI functionality")

	// ErrProviderNotFound indicates the requested provider doesn't exist
	ErrProviderNotFound = errors.New("provider not found")

	// ErrEmptyResponse indicates the AI returned no content
	ErrEmptyResponse = errors.New("provider returned empty response")
)

// Provider is the interface for AI providers that can answer questions
type Provider interface {
	// Ask sends a prompt to the AI and returns the response
	Ask(ctx context.Context, prompt string) (response string, err error)
}

// BaseProvider contains common configuration shared by all providers
type BaseProvider struct {
	// TimeoutSeconds is the maximum time to wait for AI response
	TimeoutSeconds int

	// MaxInputBytes limits the input size sent to AI (0 = no limit)
	MaxInputBytes int
}

// DefaultBaseProvider returns reasonable defaults for provider configuration
func DefaultBaseProvider() BaseProvider {
	return BaseProvider{
		TimeoutSeconds: 60,
		MaxInputBytes:  200000,
	}
}
