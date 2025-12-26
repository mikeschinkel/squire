package askai

import (
	"context"
	"time"
)

// Agent provides a higher-level interface for AI operations
// It wraps a Provider and can add functionality like retry logic, caching, etc.
type Agent struct {
	provider Provider
	timeout  time.Duration
}

// AgentArgs contains configuration for creating an Agent
type AgentArgs struct {
	// Provider is the AI provider to use
	Provider Provider

	// TimeoutSeconds is the maximum time to wait for responses (0 = no timeout)
	TimeoutSeconds int
}

// NewAgent creates a new Agent with the given provider
func NewAgent(args AgentArgs) *Agent {
	timeout := time.Duration(0)
	if args.TimeoutSeconds > 0 {
		timeout = time.Duration(args.TimeoutSeconds) * time.Second
	}

	return &Agent{
		provider: args.Provider,
		timeout:  timeout,
	}
}

// Ask sends a prompt to the AI and returns the response
func (a *Agent) Ask(ctx context.Context, prompt string) (response string, err error) {
	// Apply timeout if configured
	if a.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.timeout)
		defer cancel()
	}

	// Delegate to provider
	response, err = a.provider.Ask(ctx, prompt)
	if err != nil {
		err = NewErr(ErrAskAI, "operation", "ask", err)
		goto end
	}

end:
	return response, err
}
