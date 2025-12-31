package precommit

import (
	"context"
	"os"
	"time"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
	"github.com/mikeschinkel/gomion/gommod/goutils"
)

// Analyze performs pre-commit analysis on staged changes
// This function demonstrates the architecture:
// - Direct function calls (no pluggable interface for execution)
// - Bespoke result handling (accessing specific fields)
// - Generic formatting (using AnalysisResult interface)
func Analyze(ctx context.Context, args AnalyzeArgs) (result Results, err error) {
	var repo *gitutils.Repo
	var cachedWT *gitutils.CachedWorktree
	var stagedDir dt.DirPath
	var tempDir string

	result.Timestamp = time.Now()

	// Open repo
	repo, err = gitutils.Open(args.ModuleDir)
	if err != nil {
		goto end
	}

	// Find baseline tag
	result.BaselineTag, err = repo.FindBaselineTag(ctx, "")
	if err != nil {
		// Not fatal - set verdict to unknown and continue
		result.OverallVerdict = goutils.VerdictUnknown
		// Continue without baseline analysis
		err = nil
		goto end
	}

	// Open cached worktree and checkout baseline
	cachedWT, err = repo.OpenCachedWorktree(ctx)
	if err != nil {
		goto end
	}
	defer cachedWT.Close()

	err = cachedWT.Checkout(result.BaselineTag)
	if err != nil {
		goto end
	}

	// Create temp directory for staged files
	tempDir, err = os.MkdirTemp("", "gomion-precommit-*")
	if err != nil {
		goto end
	}
	stagedDir = dt.DirPath(tempDir)
	defer os.RemoveAll(string(stagedDir))

	// Export staged files to temp directory
	err = gitutils.ExportStagedFiles(ctx, gitutils.ExportStagedArgs{
		Repo:    repo,
		DestDir: stagedDir,
	})
	if err != nil {
		goto end
	}

	// Call each analysis function directly (bespoke handling)
	// This demonstrates: NO generic loop, direct function calls with specific types

	result.API, err = goutils.AnalyzeAPICompatibility(ctx, cachedWT.Dir, stagedDir)
	if err != nil {
		// Log but continue with other analyzers
		// In production, would log: logger.Warn("API analysis failed", "error", err)
		err = nil
	}

	result.AST, err = goutils.AnalyzeASTDiff(ctx, cachedWT.Dir, stagedDir)
	if err != nil {
		// Log but continue
		err = nil
	}

	result.Tests, err = goutils.AnalyzeTestSignals(ctx, cachedWT.Dir, stagedDir)
	if err != nil {
		// Log but continue
		err = nil
	}

	// Compute overall verdict using bespoke logic
	result.OverallVerdict = computeOverallVerdict(&result)

	// TODO: Persist results for exit/reenter workflows
	// err = persistResult(&result, args.CacheKey)

end:
	return result, err
}

// computeOverallVerdict uses bespoke logic to determine overall verdict
// This demonstrates accessing specific fields of bespoke result types
func computeOverallVerdict(r *Results) goutils.VerdictType {
	// Breaking takes precedence
	if r.API.Verdict == goutils.VerdictBreaking {
		return goutils.VerdictBreaking
	}
	if r.AST.Verdict == goutils.VerdictBreaking {
		return goutils.VerdictBreaking
	}

	// If all are likely compatible
	if r.API.Verdict == goutils.VerdictLikelyCompatible &&
		r.AST.Verdict == goutils.VerdictLikelyCompatible {
		return goutils.VerdictLikelyCompatible
	}

	// Default to maybe compatible
	return goutils.VerdictMaybeCompatible
}

// FormatForAI generates markdown for AI prompts
// This demonstrates generic formatting using the AnalysisResult interface
func (r Results) FormatForAI() string {
	var combined string

	// Use AnalysisResult interface for generic iteration
	// This is where the interface adds value - same formatting code for all analyzers
	for _, analyzer := range []goutils.AnalysisResult{r.API, r.AST, r.Tests} {
		combined += analyzer.AnalysisSummary(goutils.MarkdownFormat)
		combined += "\n\n"
	}

	return combined
}

// FormatForTerminal generates ANSI-escaped output for terminal display
// This also demonstrates generic formatting
func (r Results) FormatForTerminal() string {
	var combined string

	// Same generic loop, different format
	for _, analyzer := range []goutils.AnalysisResult{r.API, r.AST, r.Tests} {
		combined += analyzer.AnalysisSummary(goutils.ANSIEscapedFormat)
		combined += "\n"
	}

	return combined
}
