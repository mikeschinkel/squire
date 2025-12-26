package commitmsg

import (
	"errors"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/squire/squirepkg/precommit"
)

// Sentinel errors
var (
	// ErrCommitMsg is the base sentinel for all commitmsg package errors
	ErrCommitMsg = errors.New("error generating commit message")

	// ErrEmptySubject indicates the AI returned a message with no subject
	ErrEmptySubject = errors.New("empty commit subject")
)

// Request contains git-specific information needed to generate a commit message
type Request struct {
	// ModuleDir is the directory of the Go module
	ModuleDir dt.DirPath

	// Branch is the current git branch name
	Branch string

	// StagedDiff is the full git diff of staged changes
	StagedDiff string

	// StagedFiles is the list of staged file paths
	StagedFiles []string

	// ConventionalCommits requests conventional commit format (type: subject)
	ConventionalCommits bool

	// MaxSubjectChars limits the subject line length (0 = no limit)
	MaxSubjectChars int

	// AnalysisResults contains pre-commit analysis results (optional)
	AnalysisResults *precommit.Results
}

// Result contains the parsed commit message components
type Result struct {
	// Subject is the commit message subject line
	Subject string

	// Body is the commit message body (can be empty)
	Body string

	// Raw is the raw AI response (for debugging)
	Raw string
}

// Message returns the full commit message (subject + body)
func (r Result) Message() string {
	if r.Body == "" {
		return r.Subject
	}
	return r.Subject + "\n\n" + r.Body
}
