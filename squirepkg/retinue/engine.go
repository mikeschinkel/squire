package retinue

import (
	"context"
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/squire/squirepkg/apidiffr"
	"github.com/mikeschinkel/squire/squirepkg/gitutils"
)

// VerdictType represents the confidence level of a breaking change assessment
type VerdictType string

const (
	VerdictBreaking         VerdictType = "breaking"
	VerdictLikelyBreaking   VerdictType = "likely_breaking"
	VerdictMaybeNotBreaking VerdictType = "maybe_not_breaking"
	VerdictWithheld         VerdictType = "withheld"
	VerdictUnknown          VerdictType = "unknown"
)

// EngineResult contains the output of the release planning engine
type EngineResult struct {
	// LeafModuleDir is the directory of the selected leaf module
	LeafModuleDir ModuleDir

	// LeafRepoDir is the repository directory containing the leaf module
	LeafRepoDir RepoDir

	// LeafRepoModules lists all modules in the leaf repository (for informational purposes)
	LeafRepoModules []ModuleDir

	// LocalTagNotPushed indicates if there's a local tag that hasn't been pushed to remote
	LocalTagNotPushed bool

	// LocalTagNotPushedWarning contains the warning message if LocalTagNotPushed is true
	LocalTagNotPushedWarning string

	// MissingRemoteTags lists tags that exist on remote but not locally (e.g., created by GitHub Actions)
	MissingRemoteTags []string

	// Verdict is the assessment of whether this release contains breaking changes
	Verdict VerdictType

	// VerdictReason explains why this verdict was reached
	VerdictReason string

	// InFluxDependencies lists any dependencies that are in-flux (for debugging/info)
	InFluxDependencies []ModulePath

	// Git status information for the leaf repo
	Branch         gitutils.GitRef    // Current branch name
	Remote         gitutils.GitRemote // Remote; name & branch (e.g., "origin", "main")
	Ahead          int                // Commits ahead of upstream
	Behind         int                // Commits behind upstream
	StagedFiles    int                // Number of staged files
	UnstagedFiles  int                // Number of unstaged files
	UntrackedFiles int                // Number of untracked files

	// Git status information for the starting repo
	StartBranch gitutils.GitRef    // Current branch of starting repo
	StartRemote gitutils.GitRemote // Remote name of starting repo
}

// EngineArgs contains the input parameters for the release planning engine
type EngineArgs struct {
	// StartDir is the directory to start scanning from (typically current directory or repo root)
	StartDir string

	// RepoDirs is the list of repository directories to scan for modules
	RepoDirs []string

	// Config is the squire configuration
	Config *Config

	// Logger for diagnostic output
	Logger *slog.Logger

	// Writer for progress output (optional, for streaming)
	Writer cliutil.Writer
}

// StreamingHook is an optional callback for progress updates during long operations
type StreamingHook func(message string)

// ReleaseEngine is the core engine that selects a leaf module and computes verdict
type ReleaseEngine struct {
	args  EngineArgs
	graph *GoModGraph
	hook  StreamingHook
}

// NewReleaseEngine creates a new release planning engine
func NewReleaseEngine(args EngineArgs) *ReleaseEngine {
	return &ReleaseEngine{
		args: args,
	}
}

// WithStreamingHook adds an optional callback for progress updates
func (e *ReleaseEngine) WithStreamingHook(hook StreamingHook) *ReleaseEngine {
	e.hook = hook
	return e
}

// Run executes the engine and returns the result
func (e *ReleaseEngine) Run(ctx context.Context) (result *EngineResult, err error) {
	var repoDir dt.DirPath
	var repoDirsToScan []dt.DirPath
	var goModFiles []dt.Filepath
	var leafModuleDir ModuleDir
	var repo *gitutils.Repo

	result = &EngineResult{
		Verdict: VerdictUnknown,
	}

	// Step 1: Normalize and validate start directory
	startDir := e.args.StartDir
	if startDir == "" {
		startDir = "."
	}

	e.stream("Resolving start directory...")
	repoDir, err = dt.ParseDirPath(startDir)
	if err != nil {
		goto end
	}

	repoDir, err = repoDir.Clean().Abs()
	if err != nil {
		goto end
	}

	// Ensure we're in a repository
	repoDir, err = FindRepoRoot(repoDir)
	if err != nil {
		goto end
	}

	// Step 1.5: Open Git repo for starting repo
	repo, err = gitutils.Open(repoDir)
	if err != nil {
		// Not a git repo - skip
		err = nil
		goto end
	}

	result.StartBranch = repo.Branch
	result.StartRemote = repo.Remote

	// Step 2: Get list of all repos to scan
	e.stream("Scanning for repositories...")
	repoDirsToScan, err = e.getRepoDirsToScan()
	if err != nil {
		goto end
	}

	// Always include the start repo in the scan
	repoDirsToScan = append([]dt.DirPath{repoDir}, repoDirsToScan...)

	// Step 3: Find all go.mod files
	e.stream("Finding go.mod files...")
	goModFiles, err = FindGoModFiles[dt.Filepath](FindGoModFilesArgs{
		DirPaths:       repoDirsToScan,
		Config:         e.args.Config,
		ContinueOnErr:  false,
		SilenceErrs:    false,
		SkipBehavior:   SkipUnmanaged,
		MatchBehavior:  dtx.CollectOnMatch,
		ParseEntryFunc: nil,
		Logger:         e.args.Logger,
		Writer:         e.args.Writer,
	})
	if err != nil {
		goto end
	}

	//// Sort go.mod files for deterministic graph building
	//sort.Slice(goModFiles, func(i, j int) bool {
	//	return string(goModFiles[i]) < string(goModFiles[j])
	//})

	// Step 4: Build the module dependency graph
	e.stream("Building module dependency graph...")
	e.graph = NewGoModuleGraph(repoDir, goModFiles, GoModuleGraphArgs{
		Logger: e.args.Logger,
		Writer: e.args.Writer,
	})
	err = e.graph.Build()
	if err != nil {
		goto end
	}

	// Step 5: Find leaf module with no in-flux dependencies
	e.stream("Finding leaf module...")
	leafModuleDir, err = e.findLeafModule(ctx)
	if err != nil {
		goto end
	}

	// Populate result with leaf information
	result.LeafModuleDir = leafModuleDir
	result.LeafRepoDir, err = FindRepoRoot(leafModuleDir)
	if err != nil {
		goto end
	}

	// Get all modules in the leaf repo for informational purposes
	result.LeafRepoModules = e.getRepoModules(result.LeafRepoDir)

	// Step 6: Gather git status information
	e.stream("Gathering git status...")
	err = e.gatherGitStatus(ctx, result)
	if err != nil {
		goto end
	}

	// Step 7: Check for tagged-but-not-pushed
	e.stream("Checking tag/push status...")
	err = e.checkTaggedButNotPushed(ctx, result)
	if err != nil {
		goto end
	}

	// Step 8: Compute verdict using apidiffr
	e.stream("Computing breaking change verdict...")
	err = e.computeVerdict(ctx, result)
	if err != nil {
		goto end
	}

end:
	return result, err
}

// stream sends a progress message if a streaming hook is registered
func (e *ReleaseEngine) stream(message string) {
	if e.hook != nil {
		e.hook(message)
	}
}

// getRepoDirsToScan returns the list of repository directories to scan
func (e *ReleaseEngine) getRepoDirsToScan() (dirs []dt.DirPath, err error) {
	dirs, err = dt.ParseDirPaths(e.args.RepoDirs)
	if len(e.args.RepoDirs) == 0 {
		dirs = e.args.Config.ScanDirs
	}
	if len(dirs) == 0 {
		var home dt.DirPath
		home, err = dt.UserHomeDir()
		if err != nil {
			goto end
		}
		dirs = []dt.DirPath{home}
	}
end:
	return dirs, err
}

// getRepoModules returns all modules in the specified repository
func (e *ReleaseEngine) getRepoModules(repoDir RepoDir) (modules []ModuleDir) {
	if e.graph == nil {
		return nil
	}

	mods, ok := e.graph.ModulesMapByModulePathByRepoDir[repoDir]
	if !ok {
		return nil
	}

	modules = make([]ModuleDir, 0, mods.Len())
	for mod := range mods.Values() {
		modules = append(modules, mod.Dir())
	}
	return modules
}

// gatherGitStatus populates git status information in the result
func (e *ReleaseEngine) gatherGitStatus(ctx context.Context, result *EngineResult) (err error) {
	var repo *gitutils.Repo
	var upstreamState gitutils.UpstreamState
	var counts gitutils.StatusCounts
	var modRelPath dt.PathSegments
	var excludePaths []dt.PathSegments
	var missingTags []string

	// Open the git repository
	repo, err = gitutils.Open(result.LeafRepoDir)
	if err != nil {
		// Not a git repo - skip gathering status
		err = nil
		goto end
	}

	result.Branch = repo.Branch
	result.Remote = repo.Remote

	// Get ahead/behind counts
	upstreamState, err = repo.UpstreamState()
	if err != nil {
		// No upstream - not fatal, skip it
		err = nil
	} else {
		if upstreamState.Ahead() > 0 {
			result.Ahead = upstreamState.Ahead()
		}
		if upstreamState.Behind() > 0 {
			result.Behind = upstreamState.Behind()
		}
	}

	// Check for tags on remote that don't exist locally, and fetch them if found
	modRelPath, err = result.LeafModuleDir.Rel(result.LeafRepoDir)
	if err == nil {
		missingTags, err = repo.CompareRemoteTags(ctx, string(modRelPath))
		if err == nil && len(missingTags) > 0 {
			// Auto-fetch missing tags
			e.stream("Fetching missing tags from remote...")
			fetchErr := repo.FetchTags(ctx)
			if fetchErr == nil {
				// Successfully fetched - record what was fetched
				result.MissingRemoteTags = missingTags
			} else {
				// Fetch failed - log but don't fail the whole operation
				if e.args.Logger != nil {
					e.args.Logger.Warn("Failed to fetch tags", "error", fetchErr)
				}
			}
		}
		// Don't fail on tag comparison errors
		err = nil
	}

	// Calculate module relative path within repo
	modRelPath, err = result.LeafModuleDir.Rel(result.LeafRepoDir)
	if err != nil {
		// Can't determine relative path - fall back to full repo status
		counts, err = repo.StatusCounts()
	} else {
		// Build list of submodule paths to exclude
		excludePaths = e.getSubmodulePathsToExclude(result.LeafRepoDir, result.LeafModuleDir)

		// Get file status counts for the specific module directory, excluding submodules
		counts, err = repo.StatusCountsInPathExcluding(modRelPath, excludePaths)
	}
	if err != nil {
		// Can't get status - not fatal, skip it
		err = nil
	} else {
		result.StagedFiles = counts.Staged
		result.UnstagedFiles = counts.Unstaged
		result.UntrackedFiles = counts.Untracked
	}

end:
	return err
}

// getSubmodulePathsToExclude returns paths of other modules in the same repo that should be excluded
func (e *ReleaseEngine) getSubmodulePathsToExclude(repoDir RepoDir, moduleDir ModuleDir) (excludePaths []dt.PathSegments) {
	if e.graph == nil {
		return nil
	}

	// Get all modules in this repo
	repoModules := e.getRepoModules(repoDir)

	for _, otherModDir := range repoModules {
		// Skip the module we're checking
		if otherModDir == moduleDir {
			continue
		}

		// Check if this other module is a subdirectory of our module
		relPath, err := otherModDir.Rel(moduleDir)
		if err != nil {
			// Not a subdirectory, skip
			continue
		}

		// This is a submodule that should be excluded
		excludePaths = append(excludePaths, relPath)
	}

	return excludePaths
}

// findLeafModule finds a module with no in-flux dependencies
// Returns the first module found whose dependencies are not in-flux
func (e *ReleaseEngine) findLeafModule(ctx context.Context) (leafDir ModuleDir, err error) {
	var traverseResult *TraverseResult
	var leafCandidates []ModuleDir

	// Get the repo we're starting from
	_, ok := e.graph.ReposByRepoDir[e.graph.RepoDir]
	if !ok {
		err = NewErr(ErrNoGoModuleFound, "repo", e.graph.RepoDir)
		goto end
	}

	// Traverse the dependency tree in post-order (dependencies first)
	traverseResult, err = e.graph.Traverse()
	if err != nil {
		goto end
	}
	// Iterate through modules in dependency order (deepest first)
	// Collect ALL in-flux modules whose dependencies are clean (not in-flux)
	for _, moduleDirs := range traverseResult.RepoModules.Iterator() {
		for _, modDir := range moduleDirs {
			var inFlux bool
			var depsInFlux bool

			module, ok := e.graph.ModulesByModuleDir[modDir]
			if !ok {
				continue
			}

			// Check if this module is in-flux
			inFlux, err = e.isModuleInFlux(ctx, module)
			if err != nil {
				// Log error but continue searching
				if e.args.Logger != nil {
					e.args.Logger.Warn("Error checking in-flux status", "module", modDir, "error", err)
				}
				continue
			}

			if !inFlux {
				// Module is clean, skip it (already released)
				continue
			}

			// Module is in-flux, check if its dependencies are also in-flux
			depsInFlux, err = e.hasDependenciesInFlux(ctx, module)
			if err != nil {
				// Log error but continue searching
				if e.args.Logger != nil {
					e.args.Logger.Warn("Error checking dependencies in-flux status", "module", modDir, "error", err)
				}
				continue
			}

			if !depsInFlux {
				// Found a leaf candidate! Module is in-flux but all dependencies are clean
				leafCandidates = append(leafCandidates, modDir)
			}
		}
	}

	// If we found multiple leaf candidates, pick the alphabetically first one for determinism
	if len(leafCandidates) > 0 {
		leafDir = leafCandidates[0]
		goto end
	}

	// If we get here, no suitable leaf was found
	err = NewErr(ErrNoGoModuleFound,
		"reason", "no in-flux modules found with clean dependencies",
		"repo", e.graph.RepoDir,
	)

end:
	return leafDir, err
}

// isModuleInFlux checks if a module's dependencies are in-flux
func (e *ReleaseEngine) isModuleInFlux(ctx context.Context, module *GoModule) (inFlux bool, err error) {
	var reason string

	// Use the new GoModule.IsInFlux() wrapper method
	inFlux, reason, err = module.IsInFlux(ctx)
	if err != nil {
		goto end
	}

	// Log reason if in-flux and logger available
	if inFlux && e.args.Logger != nil {
		e.args.Logger.Debug("Module is in-flux",
			"module", module.Dir(),
			"reason", reason)
	}

end:
	return inFlux, err
}

// hasDependenciesInFlux checks if any of this module's dependencies are in-flux
func (e *ReleaseEngine) hasDependenciesInFlux(ctx context.Context, module *GoModule) (hasInFluxDeps bool, err error) {
	var requireDirs []ModuleDir

	// Get the module directories that this module depends on
	requireDirs = module.RequireDirs()

	for _, depDir := range requireDirs {
		var depModule *GoModule
		var ok bool
		var inFlux bool

		depModule, ok = e.graph.ModulesByModuleDir[depDir]
		if !ok {
			// Dependency not in our graph (external/not local), skip it
			continue
		}

		// Check if this dependency is in-flux
		inFlux, err = e.isModuleInFlux(ctx, depModule)
		if err != nil {
			goto end
		}

		if inFlux {
			// Found an in-flux dependency
			hasInFluxDeps = true
			goto end
		}
	}
	// All dependencies are clean
	hasInFluxDeps = false

end:
	return hasInFluxDeps, err
}

// checkTaggedButNotPushed checks if there's a local tag that hasn't been pushed
func (e *ReleaseEngine) checkTaggedButNotPushed(ctx context.Context, result *EngineResult) (err error) {
	var repo *gitutils.Repo
	var upstreamState gitutils.UpstreamState

	// Open the git repository
	repo, err = gitutils.Open(result.LeafRepoDir)
	if err != nil {
		// Not a git repo or can't access - just skip the check
		result.LocalTagNotPushed = false
		err = nil
		goto end
	}

	// Check upstream state
	upstreamState, err = repo.UpstreamState()
	if err != nil {
		// No upstream configured - skip check
		result.LocalTagNotPushed = false
		err = nil
		goto end
	}

	// If we're ahead of upstream, we might have local commits/tags not pushed
	if upstreamState.Ahead() > 0 {
		// TODO: More sophisticated check - look for tags at HEAD that don't exist on remote
		// For now, just set warning but don't block (this check is too aggressive)
		result.LocalTagNotPushed = false // Disabled for now - needs proper tag checking
		result.LocalTagNotPushedWarning = "Local branch is ahead of upstream. Didn't you mean to push?"
	}

end:
	return err
}

// computeVerdict analyzes the module and determines if there are breaking changes
func (e *ReleaseEngine) computeVerdict(ctx context.Context, result *EngineResult) (err error) {
	var repo *gitutils.Repo
	var headSHA string
	var baselineTag string
	var modRelPath dt.PathSegments
	var report apidiffr.Report
	var cached *gitutils.CachedWorktree
	var baselineModuleDir dt.DirPath
	var currentModuleDir dt.DirPath
	var inFlux bool

	// Check if module has in-flux dependencies - if so, withhold verdict
	module, ok := e.graph.ModulesByModuleDir[result.LeafModuleDir]
	if !ok {
		result.Verdict = VerdictWithheld
		result.VerdictReason = "module not found in graph"
		goto end
	}

	inFlux, err = e.isModuleInFlux(ctx, module)
	if err != nil {
		result.Verdict = VerdictWithheld
		result.VerdictReason = "error checking in-flux status: " + err.Error()
		err = nil // Don't fail, just withhold verdict
		goto end
	}

	if inFlux {
		result.Verdict = VerdictWithheld
		result.VerdictReason = "module is in-flux (clean it up before verdict can be assessed)"
		goto end
	}

	// Open git repository
	repo, err = gitutils.Open(result.LeafRepoDir)
	if err != nil {
		result.Verdict = VerdictWithheld
		result.VerdictReason = "not a git repository"
		err = nil
		goto end
	}

	// Get HEAD SHA
	headSHA, err = repo.RevParse("HEAD")
	if err != nil {
		result.Verdict = VerdictWithheld
		result.VerdictReason = "cannot determine HEAD: " + err.Error()
		err = nil
		goto end
	}

	// Calculate module relative path within repo
	modRelPath, err = result.LeafModuleDir.Rel(result.LeafRepoDir)
	if err != nil {
		result.Verdict = VerdictWithheld
		result.VerdictReason = "cannot determine module path relative to repo"
		err = nil
		goto end
	}

	// Find the latest reachable semver tag for this module
	baselineTag, err = repo.LatestTag(ctx, headSHA, &gitutils.LatestTagArgs{
		ModuleRelPath: modRelPath,
	})
	if err != nil {
		// No baseline tag - this might be the first release
		result.Verdict = VerdictWithheld
		result.VerdictReason = "no baseline tag found (first release?)"
		err = nil
		goto end
	}

	// Open a cached worktree for analysis
	cached, err = repo.OpenCachedWorktree(ctx)
	if err != nil {
		result.Verdict = VerdictWithheld
		result.VerdictReason = "cannot create cached worktree: " + err.Error()
		err = nil
		goto end
	}
	defer dt.CloseOrLog(cached)

	// Checkout baseline tag
	err = cached.Checkout(baselineTag)
	if err != nil {
		result.Verdict = VerdictWithheld
		result.VerdictReason = "cannot checkout baseline tag: " + err.Error()
		err = nil
		goto end
	}

	// Get module directory in cached worktree
	baselineModuleDir = dt.DirPathJoin(cached.Dir, modRelPath)

	// Checkout HEAD
	err = cached.Checkout(headSHA)
	if err != nil {
		result.Verdict = VerdictWithheld
		result.VerdictReason = "cannot checkout HEAD: " + err.Error()
		err = nil
		goto end
	}

	// Get module directory at HEAD
	currentModuleDir = dt.DirPathJoin(cached.Dir, modRelPath)

	// Run API diff
	report, err = apidiffr.DiffDirs(baselineModuleDir, currentModuleDir, apidiffr.CompareOptions{
		ExcludeInternalPackages: true,
	})
	if err != nil {
		result.Verdict = VerdictWithheld
		result.VerdictReason = "API diff failed: " + err.Error()
		err = nil
		goto end
	}

	// Determine verdict based on breaking changes
	if report.HasBreakingChanges() {
		result.Verdict = VerdictBreaking
		result.VerdictReason = "API analysis detected breaking changes"
	} else {
		result.Verdict = VerdictMaybeNotBreaking
		result.VerdictReason = "no breaking changes detected in exported API"
	}

end:
	return err
}
