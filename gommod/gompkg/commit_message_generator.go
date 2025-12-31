package gompkg

import (
	"bytes"
	"context"
	"embed"
	"strings"
	"text/template"

	"github.com/mikeschinkel/gomion/gommod/askai"
	"github.com/mikeschinkel/gomion/gommod/goutils"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// GenerateCommitMessage generates a commit message using the provided AI agent
func GenerateCommitMessage(ctx context.Context, agent *askai.Agent, req CommitMessageRequest) (result CommitMessageResponse, err error) {
	var prompt string
	var response string

	// Build the AI prompt from git data using templates
	prompt, err = buildCommitMessagePrompt(req)
	if err != nil {
		err = NewErr(ErrCommitMsg, "operation", "build_prompt", err)
		goto end
	}

	// Ask the AI
	response, err = agent.Ask(ctx, prompt)
	if err != nil {
		err = NewErr(ErrCommitMsg, "operation", "ask_ai", err)
		goto end
	}

	// Parse the response into commit message components
	result, err = parseCommitMessageResult(response)
	if err != nil {
		goto end
	}

end:
	return result, err
}

// buildCommitMessagePrompt constructs an AI prompt from git information using templates
func buildCommitMessagePrompt(req CommitMessageRequest) (prompt string, err error) {
	var tmplName string
	var tmpl *template.Template
	var buf bytes.Buffer
	var analysisMarkdown string
	var data struct {
		ConventionalCommits bool
		MaxSubjectChars     int
		Branch              string
		StagedDiff          string
		AnalysisResults     string
	}

	// Select template based on analysis results
	tmplName = "templates/default.tmpl"
	if req.AnalysisResults != nil {
		if req.AnalysisResults.OverallVerdict == goutils.VerdictBreaking {
			tmplName = "templates/breaking.tmpl"
		}
		// Format analysis results as markdown for AI
		analysisMarkdown = req.AnalysisResults.FormatForAI()
	}

	// Load and parse template
	tmpl, err = template.ParseFS(templatesFS, tmplName)
	if err != nil {
		goto end
	}

	// Populate template data
	data.ConventionalCommits = req.ConventionalCommits
	data.MaxSubjectChars = req.MaxSubjectChars
	data.Branch = req.Branch
	data.StagedDiff = req.StagedDiff
	data.AnalysisResults = analysisMarkdown

	err = tmpl.Execute(&buf, data)
	if err != nil {
		goto end
	}

	prompt = buf.String()

end:
	return prompt, err
}

// parseCommitMessageResult parses an AI response into commit message components
func parseCommitMessageResult(response string) (result CommitMessageResponse, err error) {
	var parts []string

	// Store raw response
	result.Raw = response

	// Trim whitespace
	response = strings.TrimSpace(response)
	if response == "" {
		err = NewErr(ErrCommitMsg, ErrEmptySubject,
			"message", "AI returned empty response")
		goto end
	}

	// Split into subject and body at first blank line
	parts = strings.SplitN(response, "\n\n", 2)
	result.Subject = strings.TrimSpace(parts[0])

	if len(parts) > 1 {
		result.Body = strings.TrimSpace(parts[1])
	}

	// Validate subject exists
	if result.Subject == "" {
		err = NewErr(ErrCommitMsg, ErrEmptySubject,
			"message", "AI returned response with no subject line")
		goto end
	}

end:
	return result, err
}
