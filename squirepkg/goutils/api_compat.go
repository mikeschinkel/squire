package goutils

import (
	"context"
	"strings"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
)

// APIChange represents a single API change
type APIChange struct {
	Type        string // "removed", "added", "modified"
	Entity      string // "func", "method", "type", "field"
	Signature   string // Full signature
	Description string // Human-readable description
}

// APICompatResult contains API compatibility analysis results
type APICompatResult struct {
	Verdict         VerdictType
	BaselineTag     string
	BreakingChanges []APIChange
	Additions       []APIChange
	Modifications   []APIChange
}

// AnalyzeAPICompatibility analyzes API compatibility between baseline and staged code
func AnalyzeAPICompatibility(ctx context.Context, baseline, staged dt.DirPath) (result APICompatResult, err error) {
	// For now, return a skeleton result
	result = APICompatResult{
		Verdict:         VerdictUnknown,
		BaselineTag:     "",
		BreakingChanges: []APIChange{},
		Additions:       []APIChange{},
		Modifications:   []APIChange{},
	}

	// Placeholder: This will call Compare() to analyze API changes
	// and categorize them into breaking/additions/modifications

	return result, err
}

// AnalysisSummary implements AnalysisResult interface
func (r APICompatResult) AnalysisSummary(format OutputFormat) string {
	switch format {
	case MarkdownFormat:
		return r.formatAsMarkdown()
	case ANSIEscapedFormat:
		return r.formatAsANSI()
	case TextFormat:
		return r.formatAsPlainText()
	default:
		return r.formatAsPlainText()
	}
}

func (r APICompatResult) formatAsMarkdown() string {
	var b strings.Builder

	b.WriteString("## API Compatibility Analysis\n\n")

	if r.BaselineTag != "" {
		dtx.Fprintf(&b, "**Baseline:** %s\n", r.BaselineTag)
	}
	dtx.Fprintf(&b, "**Verdict:** %s\n\n", r.Verdict)

	if len(r.BreakingChanges) > 0 {
		b.WriteString("### Breaking Changes\n")
		for _, change := range r.BreakingChanges {
			dtx.Fprintf(&b, "- `%s` - %s\n", change.Signature, change.Description)
		}
		b.WriteString("\n")
	}

	if len(r.Additions) > 0 {
		b.WriteString("### Non-Breaking Changes\n")
		for _, change := range r.Additions {
			dtx.Fprintf(&b, "- `%s` - %s\n", change.Signature, change.Description)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (r APICompatResult) formatAsANSI() string {
	var b strings.Builder

	// ANSI color codes
	const (
		red    = "\033[31m"
		green  = "\033[32m"
		yellow = "\033[33m"
		bold   = "\033[1m"
		reset  = "\033[0m"
	)

	dtx.Fprintf(&b, "%s%sAPI Compatibility%s\n\n", bold, green, reset)

	if r.BaselineTag != "" {
		dtx.Fprintf(&b, "Baseline: %s\n", r.BaselineTag)
	}

	verdictColor := yellow
	if r.Verdict == VerdictBreaking {
		verdictColor = red
	} else if r.Verdict == VerdictLikelyCompatible {
		verdictColor = green
	}
	dtx.Fprintf(&b, "Verdict: %s%s%s\n\n", verdictColor, r.Verdict, reset)

	if len(r.BreakingChanges) > 0 {
		dtx.Fprintf(&b, "%s%sBreaking Changes:%s\n", bold, red, reset)
		for _, change := range r.BreakingChanges {
			dtx.Fprintf(&b, "  %s• %s - %s%s\n", red, change.Signature, change.Description, reset)
		}
		b.WriteString("\n")
	}

	if len(r.Additions) > 0 {
		dtx.Fprintf(&b, "%s%sNon-Breaking Changes:%s\n", bold, green, reset)
		for _, change := range r.Additions {
			dtx.Fprintf(&b, "  %s• %s - %s%s\n", green, change.Signature, change.Description, reset)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (r APICompatResult) formatAsPlainText() string {
	var b strings.Builder

	b.WriteString("API Compatibility Analysis\n\n")

	if r.BaselineTag != "" {
		dtx.Fprintf(&b, "Baseline: %s\n", r.BaselineTag)
	}
	dtx.Fprintf(&b, "Verdict: %s\n\n", r.Verdict)

	if len(r.BreakingChanges) > 0 {
		b.WriteString("Breaking Changes:\n")
		for _, change := range r.BreakingChanges {
			dtx.Fprintf(&b, "  - %s - %s\n", change.Signature, change.Description)
		}
		b.WriteString("\n")
	}

	if len(r.Additions) > 0 {
		b.WriteString("Non-Breaking Changes:\n")
		for _, change := range r.Additions {
			dtx.Fprintf(&b, "  - %s - %s\n", change.Signature, change.Description)
		}
		b.WriteString("\n")
	}

	return b.String()
}
