package askai

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

// CodexCLIProvider implements Provider using the OpenAI Codex CLI
type CodexCLIProvider struct {
	BaseProvider

	// CodexExe is the path/name of the codex executable
	CodexExe string
}

// CodexCLIProviderArgs contains configuration for Codex CLI provider
type CodexCLIProviderArgs struct {
	BaseProvider BaseProvider
	CodexExe     string
}

// NewCodexCLIProvider creates a new Codex CLI provider with the given configuration
func NewCodexCLIProvider(args CodexCLIProviderArgs) *CodexCLIProvider {
	return &CodexCLIProvider{
		BaseProvider: args.BaseProvider,
		CodexExe:     args.CodexExe,
	}
}

// DefaultCodexCLIProviderArgs returns reasonable defaults for Codex CLI provider
func DefaultCodexCLIProviderArgs() CodexCLIProviderArgs {
	return CodexCLIProviderArgs{
		BaseProvider: DefaultBaseProvider(),
		CodexExe:     "codex",
	}
}

// Ask implements the Provider interface for Codex CLI
func (p *CodexCLIProvider) Ask(ctx context.Context, prompt string) (response string, err error) {
	var cmd *exec.Cmd
	var tmpFile *os.File
	var tmpPath string
	var output []byte

	// Truncate prompt if it exceeds size limit
	if p.MaxInputBytes > 0 && len(prompt) > p.MaxInputBytes {
		prompt = prompt[:p.MaxInputBytes] + "\n\n... (input truncated) ..."
	}

	// Create temp file for output
	tmpFile, err = os.CreateTemp("", "codex-output-*.txt")
	if err != nil {
		err = NewErr(ErrAskAI, "operation", "create_temp_file", err)
		goto end
	}
	tmpPath = tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Create command: codex exec - --color never --output-last-message <tmpFile>
	// The '-' means read prompt from stdin
	cmd = exec.CommandContext(ctx, p.CodexExe, "exec", "-",
		"--color", "never",
		"--output-last-message", tmpPath)
	cmd.Stdin = bytes.NewReader([]byte(prompt))

	// Execute command
	_, err = cmd.Output()
	if err != nil {
		// Try to provide helpful error message
		if _, lookErr := exec.LookPath(p.CodexExe); lookErr != nil {
			err = NewErr(ErrAskAI, ErrProviderNotFound,
				"executable", p.CodexExe,
				"message", "codex command not found: install OpenAI Codex CLI first",
				err)
			goto end
		}
		err = NewErr(ErrAskAI,
			"executable", p.CodexExe,
			err)
		goto end
	}

	// Read response from temp file
	output, err = os.ReadFile(tmpPath)
	if err != nil {
		err = NewErr(ErrAskAI, "operation", "read_output_file", err)
		goto end
	}

	// Return trimmed response
	response = string(bytes.TrimSpace(output))
	if response == "" {
		err = NewErr(ErrAskAI, ErrEmptyResponse,
			"message", "codex returned empty response")
		goto end
	}

end:
	return response, err
}
