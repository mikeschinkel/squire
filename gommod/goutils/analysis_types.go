package goutils

// OutputFormat specifies the output format for analysis summaries
type OutputFormat string

const (
	MarkdownFormat    OutputFormat = "markdown"     // For AI prompts
	TextFormat        OutputFormat = "text"         // Plain text (logs, files)
	ANSIEscapedFormat OutputFormat = "ansi_escaped" // Terminal display (colors, bold, etc.)
)

// AnalysisResult is implemented by all analysis result types for formatting
type AnalysisResult interface {
	AnalysisSummary(format OutputFormat) string
}

// VerdictType indicates compatibility assessment
type VerdictType string

const (
	VerdictUnspecified      VerdictType = ""
	VerdictBreaking         VerdictType = "breaking"
	VerdictLikelyCompatible VerdictType = "likely-compatible" // NEVER claim absolute "compatible"
	VerdictMaybeCompatible  VerdictType = "maybe-compatible"
	VerdictUnknown          VerdictType = "unknown"
	VerdictNoChanges        VerdictType = "no-changes"
)
