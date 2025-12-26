package commitmsg

import (
	"context"
	"io"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/squire/squirepkg/askai"
	"github.com/mikeschinkel/squire/squirepkg/gitutils"
	"github.com/mikeschinkel/squire/squirepkg/precommit"
)

// GenerateWithAnalysisArgs contains arguments for GenerateWithAnalysis
type GenerateWithAnalysisArgs struct {
	ModuleDir dt.DirPath
	Logger    *slog.Logger
	Writer    io.Writer
	Agent     *askai.Agent
}

// GenerateWithAnalysis generates a commit message with pre-commit analysis
func GenerateWithAnalysis(ctx context.Context, args GenerateWithAnalysisArgs) (message string, analysisResults *precommit.Results, err error) {
	var repo *gitutils.Repo
	var stagedFiles []dt.RelFilepath
	var cacheKey string
	var cmd *exec.Cmd

	// Check if there are staged changes
	cmd = exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = string(args.ModuleDir)
	if cmd.Run() == nil {
		// Exit code 0 means no staged changes
		dtx.Fprintf(args.Writer, "No staged changes to generate commit message for.\n")
		goto end
	}

	// Run pre-commit analysis
	dtx.Fprintf(args.Writer, "Analyzing staged changes...\n")

	// Open repo to get staged files for cache key
	repo, err = gitutils.Open(args.ModuleDir)
	if err != nil {
		// Not fatal - continue without analysis
		args.Logger.Warn("Failed to open repo for analysis", "error", err)
		err = nil
		goto generate
	}

	stagedFiles, err = repo.GetStagedFiles(ctx)
	if err != nil {
		// Not fatal - continue without analysis
		args.Logger.Warn("Failed to get staged files", "error", err)
		err = nil
		goto generate
	}

	// Compute cache key and run analysis
	cacheKey = precommit.ComputeCacheKey(args.ModuleDir, stagedFiles)
	analysisResults, err = precommit.AnalyzeWithCache(ctx, args.ModuleDir, cacheKey, args.Writer)
	if err != nil {
		// Not fatal - warn and continue without analysis
		args.Logger.Warn("Pre-commit analysis failed", "error", err)
		dtx.Fprintf(args.Writer, "Warning: Analysis failed, continuing without analysis results\n\n")
		err = nil
		analysisResults = nil
	}

	// Display analysis results if available
	if analysisResults != nil {
		summary := analysisResults.FormatForTerminal()
		dtx.Fprintf(args.Writer, "%s\n", summary)
	}

generate:
	// Generate commit message
	message, err = GenerateMessage(ctx, args.ModuleDir, analysisResults, args.Agent)
	if err != nil {
		goto end
	}

	// Display the generated message
	dtx.Fprintf(args.Writer, "Generated commit message:\n")
	dtx.Fprintf(args.Writer, "─────────────────────────────────────\n")
	dtx.Fprintf(args.Writer, "%s\n", message)
	dtx.Fprintf(args.Writer, "─────────────────────────────────────\n\n")

end:
	return message, analysisResults, err
}

// GenerateMessage generates a commit message using the AI agent
func GenerateMessage(ctx context.Context, moduleDir dt.DirPath, analysisResults *precommit.Results, agent *askai.Agent) (message string, err error) {
	var diff []byte
	var cmd *exec.Cmd
	var result Result
	var branch string
	var branchBytes []byte
	var branchErr error

	// Get staged diff
	cmd = exec.Command("git", "diff", "--cached")
	cmd.Dir = string(moduleDir)
	diff, err = cmd.Output()
	if err != nil {
		goto end
	}

	// Get current branch
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = string(moduleDir)
	branchBytes, branchErr = cmd.Output()
	if branchErr == nil {
		branch = strings.TrimSpace(string(branchBytes))
	}

	// Generate commit message using commitmsg package
	result, err = GenerateCommitMessage(ctx, agent, Request{
		ModuleDir:           moduleDir,
		Branch:              branch,
		StagedDiff:          string(diff),
		StagedFiles:         []string{}, // TODO: get list of staged files
		ConventionalCommits: true,
		MaxSubjectChars:     50,
		AnalysisResults:     analysisResults,
	})
	if err != nil {
		goto end
	}

	message = result.Message()

end:
	return message, err
}

// RegenerateMessage regenerates a commit message with analysis results
func RegenerateMessage(ctx context.Context, moduleDir dt.DirPath, analysisResults *precommit.Results, agent *askai.Agent, writer io.Writer) (message string, err error) {
	dtx.Fprintf(writer, "Regenerating...\n")
	message, err = GenerateMessage(ctx, moduleDir, analysisResults, agent)
	if err != nil {
		dtx.Fprintf(writer, "Error regenerating: %v\n", err)
		goto end
	}

	dtx.Fprintf(writer, "Generated commit message:\n")
	dtx.Fprintf(writer, "─────────────────────────────────────\n")
	dtx.Fprintf(writer, "%s\n", message)
	dtx.Fprintf(writer, "─────────────────────────────────────\n\n")

end:
	return message, err
}
