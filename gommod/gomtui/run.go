package gomtui

import (
	"context"
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/gomion/gommod/askai"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
	"github.com/mikeschinkel/gomion/gommod/gompkg"
)

// TUI represents the GRU TUI staging editor
type TUI struct {
	Writer cliutil.Writer
	Logger *slog.Logger
}

// New creates a new TUI instance
func New(writer cliutil.Writer, logger *slog.Logger) *TUI {
	return &TUI{
		Writer: writer,
		Logger: logger,
	}
}

// Run is the entry point for the GRU TUI staging editor
// It takes the CLI args and launches the bubbletea app
func (t *TUI) Run(args []string) (err error) {
	var moduleDir dt.DirPath
	var userRepo *gitutils.Repo
	var cachedRepo *gitutils.CachedWorktree
	var ctx context.Context
	var exists bool

	// Parse module directory from args (defaults to current directory)
	if len(args) > 0 {
		moduleDir = dt.DirPath(args[0])
	} else {
		// Default to current directory
		moduleDir, err = dtx.GetWorkingDir()
		if err != nil {
			err = NewErr(dt.ErrCannotDetermineWorkingDirectory, err)
			goto end
		}
	}

	// Ensure module directory exists
	exists, err = moduleDir.Exists()
	if err != nil {
		err = NewErr(dt.ErrFileSystem, err)
		goto end
	}
	if !exists {
		err = NewErr(dt.ErrDirDoesNotExist)
		goto end
	}

	// Open user repository
	userRepo, err = gitutils.Open(moduleDir)
	if err != nil {
		err = NewErr(ErrGit, err)
		goto end
	}

	// Open cached worktree
	ctx = context.Background()
	cachedRepo, err = userRepo.OpenCachedWorktree(ctx)
	if err != nil {
		err = NewErr(ErrGit, err)
		goto end
	}

	// Load or generate takes
	err = t.loadOrGenerateTakes(ctx, userRepo, cachedRepo)
	if err != nil {
		goto end
	}

	// TODO: Initialize EditorState
	// TODO: Launch bubbletea program

	// For now, just print success message
	t.Writer.Printf("GRU TUI initialized successfully\n")
	t.Writer.Printf("Module: %s\n", moduleDir)
	t.Writer.Printf("Cached repo: %s\n", cachedRepo.Dir)

end:
	if err != nil {
		err = WithErr(err, "module_dir", moduleDir)
	}
	return err
}

// loadOrGenerateTakes loads cached takes or generates new ones via AI
func (t *TUI) loadOrGenerateTakes(
	ctx context.Context,
	userRepo *gitutils.Repo,
	cachedRepo *gitutils.CachedWorktree,
) (err error) {
	var changedFiles []dt.RelFilepath
	var diff string
	var cacheKey string
	var takes *gompkg.PlanTakes
	var defaultTake *gompkg.PlanTakes

	// Get changed files from git status
	changedFiles, err = t.getChangedFiles(ctx, userRepo)
	if err != nil {
		goto end
	}

	// If no changes, nothing to do
	if len(changedFiles) == 0 {
		t.Writer.Printf("No changes to stage\n")
		goto end
	}

	// Get git diff output
	diff, err = t.getGitDiff(ctx, userRepo)
	if err != nil {
		goto end
	}

	// Compute cache key based on files and diff
	cacheKey = gompkg.ComputeAnalysisCacheKey(changedFiles, diff)

	// Try to load from cache
	takes, err = gompkg.LoadPlanTakes(cacheKey)
	if err == nil {
		t.Writer.Printf("Loaded cached takes for %d files\n", len(changedFiles))
		goto end
	}

	// Cache miss - generate via AI or use default
	t.Writer.Printf("Generating takes for %d changed files...\n", len(changedFiles))

	// Try AI generation (may fail if no AI agent configured)
	takes, err = t.generateTakesViaAI(ctx, changedFiles, diff)
	if err != nil {
		// AI generation failed - use default take
		t.Writer.Printf("AI generation failed, using default take: %v\n", err)
		defaultTake = gompkg.CreateDefaultTake(changedFiles)
		takes = defaultTake
		err = nil // Clear error, default is fine
	}

	// Set cache key and save to cache
	takes.CacheKey = cacheKey
	err = gompkg.SavePlanTakes(cacheKey, takes)
	if err != nil {
		// Non-fatal - just log warning
		t.Writer.Printf("Warning: Failed to cache takes: %v\n", err)
		err = nil
	}

end:
	return err
}

// getChangedFiles returns list of changed files in working directory
func (t *TUI) getChangedFiles(ctx context.Context, repo *gitutils.Repo) (files []dt.RelFilepath, err error) {
	files, err = repo.GetChangedFiles(ctx)
	if err != nil {
		err = NewErr(ErrGit, err, "operation", "GetChangedFiles")
	}
	return files, err
}

// getGitDiff returns full git diff output for changed files
func (t *TUI) getGitDiff(ctx context.Context, repo *gitutils.Repo) (diff string, err error) {
	diff, err = repo.GetWorkingDiff(ctx)
	if err != nil {
		err = NewErr(ErrGit, err, "operation", "GetWorkingDiff")
	}
	return diff, err
}

// generateTakesViaAI calls AI to generate takes using Claude CLI
func (t *TUI) generateTakesViaAI(
	ctx context.Context,
	files []dt.RelFilepath,
	diff string,
) (takes *gompkg.PlanTakes, err error) {
	var agent *askai.Agent

	// Create agent with default Claude CLI provider
	// NewAgent defaults to ClaudeCLIProvider with "claude" executable
	agent = askai.NewAgent(askai.AgentArgs{
		TimeoutSeconds: 120, // 2 minute timeout for AI generation
	})

	// Generate takes via AI
	takes, err = gompkg.GeneratePlanTakes(ctx, agent, files, diff)
	if err != nil {
		err = NewErr(ErrGit, err, "operation", "GeneratePlanTakes")
	}

	return takes, err
}
