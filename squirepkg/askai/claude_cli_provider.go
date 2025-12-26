package askai

import (
	"bytes"
	"context"
	"os/exec"
)

// ClaudeCLIProvider implements Provider using the Claude Code CLI
type ClaudeCLIProvider struct {
	BaseProvider

	// ClaudeExe is the path/name of the claude executable
	ClaudeExe string
}

// ClaudeCLIProviderArgs contains configuration for Claude CLI provider
type ClaudeCLIProviderArgs struct {
	BaseProvider BaseProvider
	ClaudeExe    string
}

// NewClaudeCLIProvider creates a new Claude CLI provider with the given configuration
func NewClaudeCLIProvider(args ClaudeCLIProviderArgs) *ClaudeCLIProvider {
	return &ClaudeCLIProvider{
		BaseProvider: args.BaseProvider,
		ClaudeExe:    args.ClaudeExe,
	}
}

// DefaultClaudeCLIProviderArgs returns reasonable defaults for Claude CLI provider
func DefaultClaudeCLIProviderArgs() ClaudeCLIProviderArgs {
	return ClaudeCLIProviderArgs{
		BaseProvider: DefaultBaseProvider(),
		ClaudeExe:    "claude",
	}
}

// Ask implements the Provider interface for Claude CLI
func (p *ClaudeCLIProvider) Ask(ctx context.Context, prompt string) (response string, err error) {
	var out []byte
	var cmd *exec.Cmd

	// Truncate prompt if it exceeds size limit
	if p.MaxInputBytes > 0 && len(prompt) > p.MaxInputBytes {
		prompt = prompt[:p.MaxInputBytes] + "\n\n... (input truncated) ..."
	}

	// Create command with context for timeout
	cmd = exec.CommandContext(ctx, p.ClaudeExe, "-p", prompt)

	// Execute command
	out, err = cmd.Output()
	if err != nil {
		// Try to provide helpful error message
		if _, lookErr := exec.LookPath(p.ClaudeExe); lookErr != nil {
			err = NewErr(ErrAskAI, ErrProviderNotFound,
				"executable", p.ClaudeExe,
				"message", "claude command not found: install Claude Code CLI first",
				err)
			goto end
		}
		err = NewErr(ErrAskAI,
			"executable", p.ClaudeExe,
			err)
		goto end
	}

	// Return trimmed response
	response = string(bytes.TrimSpace(out))
	if response == "" {
		err = NewErr(ErrAskAI, ErrEmptyResponse,
			"message", "claude returned empty response")
		goto end
	}

end:
	return response, err
}
