//go:build localtest

package askai_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mikeschinkel/gomion/gommod/askai"
)

// containsAnyOf checks if s contains any of the substrings (case-insensitive)
func containsAnyOf(s string, substrings []string) bool {
	sLower := strings.ToLower(s)
	for _, substr := range substrings {
		if strings.Contains(sLower, strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

// providerTestCase defines a provider to test
type providerTestCase struct {
	name     string
	provider askai.Provider
	skip     bool   // Skip if provider not available
	skipMsg  string // Reason for skipping
}

// questionTestCase defines a question to ask the AI
type questionTestCase struct {
	name               string
	prompt             string
	minResponseLen     int      // Minimum expected response length
	shouldContain      string   // Response should contain this substring (case-insensitive)
	shouldContainAnyOf []string // Response should contain at least one of these (case-insensitive)
	shouldNotContain   string   // Response should NOT contain this substring
}

func TestProviders_Ask(t *testing.T) {
	// Outer table: providers to test
	providers := []providerTestCase{
		{
			name: "Claude CLI",
			provider: askai.NewClaudeCLIProvider(askai.ClaudeCLIProviderArgs{
				BaseProvider: askai.BaseProvider{
					TimeoutSeconds: 30,
					MaxInputBytes:  100000,
				},
				ClaudeExe: "claude",
			}),
			skip:    false,
			skipMsg: "claude CLI not available",
		},
		{
			name: "Codex CLI",
			provider: askai.NewCodexCLIProvider(askai.CodexCLIProviderArgs{
				BaseProvider: askai.BaseProvider{
					TimeoutSeconds: 30,
					MaxInputBytes:  100000,
				},
				CodexExe: "codex",
			}),
			skip:    false,
			skipMsg: "codex CLI not available",
		},
	}

	// Inner table: questions to ask
	questions := []questionTestCase{
		{
			name:           "Simple math question",
			prompt:         "What is 2 + 2? Answer with just the number.",
			minResponseLen: 1,
			shouldContain:  "4",
		},
		{
			name:           "Capital city question",
			prompt:         "What is the capital of France? Answer with just the city name.",
			minResponseLen: 4,
			shouldContain:  "paris",
		},
		{
			name:             "Programming language question",
			prompt:           "Name one popular programming language. Answer with just the language name.",
			minResponseLen:   2,
			shouldNotContain: "programming language that",
		},
		{
			name:               "Code explanation",
			prompt:             "Explain what this Go code does in one sentence: func Add(a, b int) int { return a + b }",
			minResponseLen:     20,
			shouldContainAnyOf: []string{"function", "two parameters", "integer", "add", "sum"},
		},
	}

	// Test each provider with each question
	for _, pc := range providers {
		t.Run(pc.name, func(t *testing.T) {
			if pc.skip {
				t.Skip(pc.skipMsg)
			}

			// Create agent with this provider
			agent := askai.NewAgent(askai.AgentArgs{
				Provider:       pc.provider,
				TimeoutSeconds: 30,
			})

			// Test each question
			for _, qc := range questions {
				t.Run(qc.name, func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()

					// Ask the question
					response, err := agent.Ask(ctx, qc.prompt)
					if err != nil {
						t.Fatalf("Ask() failed: %v", err)
					}

					// Validate response is not empty
					if response == "" {
						t.Error("Ask() returned empty response")
					}

					// Validate minimum length
					if len(response) < qc.minResponseLen {
						t.Errorf("Ask() response too short: got %d chars, want at least %d chars",
							len(response), qc.minResponseLen)
					}

					// Validate contains expected substring
					if qc.shouldContain != "" {
						responseLower := strings.ToLower(response)
						containLower := strings.ToLower(qc.shouldContain)
						if !strings.Contains(responseLower, containLower) {
							t.Errorf("Ask() response should contain %q, got: %s",
								qc.shouldContain, response)
						}
					}

					// Validate contains at least one of the expected substrings
					if len(qc.shouldContainAnyOf) > 0 {
						if !containsAnyOf(response, qc.shouldContainAnyOf) {
							t.Errorf("Ask() response should contain at least one of %v, got: %s",
								qc.shouldContainAnyOf, response)
						}
					}

					// Validate does NOT contain unwanted substring
					if qc.shouldNotContain != "" {
						responseLower := strings.ToLower(response)
						notContainLower := strings.ToLower(qc.shouldNotContain)
						if strings.Contains(responseLower, notContainLower) {
							t.Errorf("Ask() response should NOT contain %q, got: %s",
								qc.shouldNotContain, response)
						}
					}

					// Log response for inspection
					t.Logf("Response: %s", response)
				})
			}
		})
	}
}

func TestProviders_EmptyPrompt(t *testing.T) {
	providers := []providerTestCase{
		{
			name: "Claude CLI",
			provider: askai.NewClaudeCLIProvider(askai.ClaudeCLIProviderArgs{
				BaseProvider: askai.DefaultBaseProvider(),
				ClaudeExe:    "claude",
			}),
		},
		{
			name: "Codex CLI",
			provider: askai.NewCodexCLIProvider(askai.CodexCLIProviderArgs{
				BaseProvider: askai.DefaultBaseProvider(),
				CodexExe:     "codex",
			}),
		},
	}

	for _, pc := range providers {
		t.Run(pc.name, func(t *testing.T) {
			if pc.skip {
				t.Skip(pc.skipMsg)
			}

			agent := askai.NewAgent(askai.AgentArgs{
				Provider:       pc.provider,
				TimeoutSeconds: 10,
			})

			ctx := context.Background()

			// Empty prompt should still work (providers might return something)
			response, err := agent.Ask(ctx, "")
			if err == nil && response == "" {
				t.Error("Ask() with empty prompt returned empty response without error")
			}

			t.Logf("Empty prompt response: %s (err: %v)", response, err)
		})
	}
}

func TestProviders_Timeout(t *testing.T) {
	providers := []providerTestCase{
		{
			name: "Claude CLI",
			provider: askai.NewClaudeCLIProvider(askai.ClaudeCLIProviderArgs{
				BaseProvider: askai.DefaultBaseProvider(),
				ClaudeExe:    "claude",
			}),
		},
		{
			name: "Codex CLI",
			provider: askai.NewCodexCLIProvider(askai.CodexCLIProviderArgs{
				BaseProvider: askai.DefaultBaseProvider(),
				CodexExe:     "codex",
			}),
		},
	}

	for _, pc := range providers {
		t.Run(pc.name, func(t *testing.T) {
			if pc.skip {
				t.Skip(pc.skipMsg)
			}

			// Create agent with very short timeout
			agent := askai.NewAgent(askai.AgentArgs{
				Provider:       pc.provider,
				TimeoutSeconds: 1, // 1 second - very short
			})

			ctx := context.Background()

			// This might timeout or succeed depending on how fast the AI responds
			_, err := agent.Ask(ctx, "Write a very long essay about the history of computing.")

			// We don't assert anything specific - just log what happens
			// Timeout behavior can vary
			t.Logf("Timeout test result: err=%v", err)
		})
	}
}

func TestProviders_InputTruncation(t *testing.T) {
	providers := []providerTestCase{
		{
			name: "Claude CLI",
			provider: askai.NewClaudeCLIProvider(askai.ClaudeCLIProviderArgs{
				BaseProvider: askai.BaseProvider{
					TimeoutSeconds: 30,
					MaxInputBytes:  100, // Very small limit
				},
				ClaudeExe: "claude",
			}),
		},
		{
			name: "Codex CLI",
			provider: askai.NewCodexCLIProvider(askai.CodexCLIProviderArgs{
				BaseProvider: askai.BaseProvider{
					TimeoutSeconds: 30,
					MaxInputBytes:  100, // Very small limit
				},
				CodexExe: "codex",
			}),
		},
	}

	for _, pc := range providers {
		t.Run(pc.name, func(t *testing.T) {
			if pc.skip {
				t.Skip(pc.skipMsg)
			}

			agent := askai.NewAgent(askai.AgentArgs{
				Provider:       pc.provider,
				TimeoutSeconds: 30,
			})

			ctx := context.Background()

			// Create a very long prompt that will be truncated
			longPrompt := strings.Repeat("This is a very long prompt. ", 100)

			// Should still work with truncated input
			response, err := agent.Ask(ctx, longPrompt)
			if err != nil {
				t.Fatalf("Ask() with long prompt failed: %v", err)
			}

			if response == "" {
				t.Error("Ask() with truncated input returned empty response")
			}

			t.Logf("Truncation test response length: %d", len(response))
		})
	}
}
