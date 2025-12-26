package goutils

import (
	"context"
	"strings"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
)

// TypeChange represents a change to a type definition
type TypeChange struct {
	TypeName    string
	ChangeType  string // "added", "removed", "modified"
	Description string
}

// FuncChange represents a change to a function
type FuncChange struct {
	FuncName    string
	ChangeType  string // "added", "removed", "modified"
	Signature   string
	Description string
}

// ASTDiffResult contains AST-level analysis results
type ASTDiffResult struct {
	Verdict          VerdictType
	TypeChanges      []TypeChange
	FuncChanges      []FuncChange
	DocChanges       []string // Significant doc comment changes
	StructTagChanges []string // Struct tag modifications
}

// AnalyzeASTDiff analyzes AST-level changes between baseline and staged code
func AnalyzeASTDiff(ctx context.Context, baseline, staged dt.DirPath) (result ASTDiffResult, err error) {
	// TODO: Implement using go/parser and go/ast
	// For now, return a skeleton result
	result = ASTDiffResult{
		Verdict:          VerdictUnknown,
		TypeChanges:      []TypeChange{},
		FuncChanges:      []FuncChange{},
		DocChanges:       []string{},
		StructTagChanges: []string{},
	}

	// Placeholder: This will parse both directories using go/parser,
	// compare ASTs, and detect type/function/doc changes

	return result, err
}

// AnalysisSummary implements AnalysisResult interface
func (r ASTDiffResult) AnalysisSummary(format OutputFormat) string {
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

func (r ASTDiffResult) formatAsMarkdown() string {
	var b strings.Builder

	b.WriteString("## AST Analysis\n\n")
	dtx.Fprintf(&b, "**Verdict:** %s\n\n", r.Verdict)

	if len(r.TypeChanges) > 0 {
		b.WriteString("### Type Changes\n")
		for _, change := range r.TypeChanges {
			dtx.Fprintf(&b, "- `%s` - %s: %s\n", change.TypeName, change.ChangeType, change.Description)
		}
		b.WriteString("\n")
	}

	if len(r.FuncChanges) > 0 {
		b.WriteString("### Function Changes\n")
		for _, change := range r.FuncChanges {
			dtx.Fprintf(&b, "- `%s` - %s: %s\n", change.FuncName, change.ChangeType, change.Description)
		}
		b.WriteString("\n")
	}

	if len(r.DocChanges) > 0 {
		b.WriteString("### Documentation Changes\n")
		for _, change := range r.DocChanges {
			dtx.Fprintf(&b, "- %s\n", change)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (r ASTDiffResult) formatAsANSI() string {
	var b strings.Builder

	const (
		blue   = "\033[34m"
		green  = "\033[32m"
		yellow = "\033[33m"
		bold   = "\033[1m"
		reset  = "\033[0m"
	)

	dtx.Fprintf(&b, "%s%sAST Analysis%s\n\n", bold, blue, reset)
	dtx.Fprintf(&b, "Verdict: %s%s%s\n\n", yellow, r.Verdict, reset)

	if len(r.TypeChanges) > 0 {
		dtx.Fprintf(&b, "%s%sType Changes:%s\n", bold, green, reset)
		for _, change := range r.TypeChanges {
			dtx.Fprintf(&b, "  • %s - %s: %s\n", change.TypeName, change.ChangeType, change.Description)
		}
		b.WriteString("\n")
	}

	if len(r.FuncChanges) > 0 {
		dtx.Fprintf(&b, "%s%sFunction Changes:%s\n", bold, green, reset)
		for _, change := range r.FuncChanges {
			dtx.Fprintf(&b, "  • %s - %s: %s\n", change.FuncName, change.ChangeType, change.Description)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (r ASTDiffResult) formatAsPlainText() string {
	var b strings.Builder

	b.WriteString("AST Analysis\n\n")
	dtx.Fprintf(&b, "Verdict: %s\n\n", r.Verdict)

	if len(r.TypeChanges) > 0 {
		b.WriteString("Type Changes:\n")
		for _, change := range r.TypeChanges {
			dtx.Fprintf(&b, "  - %s - %s: %s\n", change.TypeName, change.ChangeType, change.Description)
		}
		b.WriteString("\n")
	}

	if len(r.FuncChanges) > 0 {
		b.WriteString("Function Changes:\n")
		for _, change := range r.FuncChanges {
			dtx.Fprintf(&b, "  - %s - %s: %s\n", change.FuncName, change.ChangeType, change.Description)
		}
		b.WriteString("\n")
	}

	return b.String()
}
