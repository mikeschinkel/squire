package precommit

import (
	"time"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/goutils"
)

// Results contains all pre-commit analysis results
type Results struct {
	Timestamp      time.Time
	BaselineTag    string
	ModulePath     string
	OverallVerdict goutils.VerdictType

	// Individual analysis results (bespoke types from goutils)
	API   goutils.APICompatResult
	AST   goutils.ASTDiffResult
	Tests goutils.TestSignalsResult
}

// AnalyzeArgs contains arguments for the Analyze function
type AnalyzeArgs struct {
	ModuleDir dt.DirPath
	CacheKey  string // For persistence
}

// CommitGroup represents a suggested grouping of files for a single commit
type CommitGroup struct {
	Title     string
	Files     []dt.RelFilepath
	Rationale string
	Suggested bool // AI suggested vs user-defined
}
