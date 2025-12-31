package goutils

import (
	"context"
	"strings"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
)

// TestSignalsResult contains test change analysis results
type TestSignalsResult struct {
	VerdictType    VerdictType
	NewTests       []string // Newly added test functions
	ModifiedTests  []string // Changed test functions
	RemovedTests   []string // Deleted test functions
	NewTestCount   int      // Count of new tests
	CoverageSignal string   // "good", "poor", "unknown"
}

// AnalyzeTestSignals detects test changes between baseline and staged code
//
//goland:noinspection GoUnusedParameter
func AnalyzeTestSignals(ctx context.Context, baseline, staged dt.DirPath) (result TestSignalsResult, err error) {
	// TODO: Implement using go/parser to find *_test.go files and test functions
	// For now, return a skeleton result
	result = TestSignalsResult{
		VerdictType:    VerdictUnknown,
		NewTests:       []string{},
		ModifiedTests:  []string{},
		RemovedTests:   []string{},
		NewTestCount:   0,
		CoverageSignal: "unknown",
	}

	// Placeholder: This will find *_test.go files in both dirs,
	// parse them, and detect new/modified/removed test functions

	return result, err
}

// AnalysisSummary implements AnalysisResult interface
func (r TestSignalsResult) AnalysisSummary(format OutputFormat) string {
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

func (r TestSignalsResult) formatAsMarkdown() string {
	var b strings.Builder

	b.WriteString("## Test Analysis\n\n")
	dtx.Fprintf(&b, "**Verdict:** %s\n\n", r.VerdictType)

	if len(r.NewTests) > 0 {
		b.WriteString("### New Tests\n")
		for _, test := range r.NewTests {
			dtx.Fprintf(&b, "- `%s` - NEW\n", test)
		}
		b.WriteString("\n")
	}

	if len(r.ModifiedTests) > 0 {
		b.WriteString("### Modified Tests\n")
		for _, test := range r.ModifiedTests {
			dtx.Fprintf(&b, "- `%s` - MODIFIED\n", test)
		}
		b.WriteString("\n")
	}

	if len(r.RemovedTests) > 0 {
		b.WriteString("### Removed Tests\n")
		for _, test := range r.RemovedTests {
			dtx.Fprintf(&b, "- `%s` - REMOVED\n", test)
		}
		b.WriteString("\n")
	}

	if r.CoverageSignal != "unknown" {
		b.WriteString("### Coverage Signals\n")
		if r.NewTestCount > 0 {
			dtx.Fprintf(&b, "- New functionality is well-tested (%d new tests added)\n", r.NewTestCount)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (r TestSignalsResult) formatAsANSI() string {
	var b strings.Builder

	const (
		cyan   = "\033[36m"
		green  = "\033[32m"
		red    = "\033[31m"
		yellow = "\033[33m"
		bold   = "\033[1m"
		reset  = "\033[0m"
	)

	dtx.Fprintf(&b, "%s%sTest Analysis%s\n\n", bold, cyan, reset)
	dtx.Fprintf(&b, "Verdict: %s%s%s\n\n", yellow, r.VerdictType, reset)

	if len(r.NewTests) > 0 {
		dtx.Fprintf(&b, "%s%sNew Tests:%s\n", bold, green, reset)
		for _, test := range r.NewTests {
			dtx.Fprintf(&b, "  %s✓ %s%s\n", green, test, reset)
		}
		b.WriteString("\n")
	}

	if len(r.ModifiedTests) > 0 {
		dtx.Fprintf(&b, "%s%sModified Tests:%s\n", bold, yellow, reset)
		for _, test := range r.ModifiedTests {
			dtx.Fprintf(&b, "  %s• %s%s\n", yellow, test, reset)
		}
		b.WriteString("\n")
	}

	if len(r.RemovedTests) > 0 {
		dtx.Fprintf(&b, "%s%sRemoved Tests:%s\n", bold, red, reset)
		for _, test := range r.RemovedTests {
			dtx.Fprintf(&b, "  %s✗ %s%s\n", red, test, reset)
		}
		b.WriteString("\n")
	}

	if r.NewTestCount > 0 {
		dtx.Fprintf(&b, "%sCoverage: %sGood (%d new tests)%s\n", bold, green, r.NewTestCount, reset)
	}

	return b.String()
}

func (r TestSignalsResult) formatAsPlainText() string {
	var b strings.Builder

	b.WriteString("Test Analysis\n\n")
	dtx.Fprintf(&b, "Verdict: %s\n\n", r.VerdictType)

	if len(r.NewTests) > 0 {
		b.WriteString("New Tests:\n")
		for _, test := range r.NewTests {
			dtx.Fprintf(&b, "  - %s\n", test)
		}
		b.WriteString("\n")
	}

	if len(r.ModifiedTests) > 0 {
		b.WriteString("Modified Tests:\n")
		for _, test := range r.ModifiedTests {
			dtx.Fprintf(&b, "  - %s\n", test)
		}
		b.WriteString("\n")
	}

	if len(r.RemovedTests) > 0 {
		b.WriteString("Removed Tests:\n")
		for _, test := range r.RemovedTests {
			dtx.Fprintf(&b, "  - %s\n", test)
		}
		b.WriteString("\n")
	}

	if r.NewTestCount > 0 {
		dtx.Fprintf(&b, "Coverage: Good (%d new tests added)\n", r.NewTestCount)
	}

	return b.String()
}
