package gompkg

import (
	"context"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
	"github.com/mikeschinkel/gomion/gommod/goutils"
)

// ModuleExt wraps goutils.Module with Gomion-specific functionality
type ModuleExt struct {
	*goutils.Module
	graph *goutils.ModuleGraph // Base graph from goutils
	repo  *goutils.Repo        // Base repo from goutils
}

// NewModuleExt creates a new ModuleExt wrapping a goutils.Module
func NewModuleExt(modFile dt.Filepath) *ModuleExt {
	return &ModuleExt{
		Module: goutils.NewModule(modFile),
	}
}

// SetGraph sets the ModuleGraph for this module
func (m *ModuleExt) SetGraph(graph *goutils.ModuleGraph) (err error) {
	var ok bool
	m.repo, ok = graph.ReposByModuleDir[m.Filepath.Dir()]
	if !ok {
		err = NewErr(ErrRepoNotFoundForGoModule,
			"module_file", m.Filepath,
		)
	}
	m.graph = graph

	// Also set the base Module's ModuleGraph
	err = m.Module.SetGraph(graph)
	return err
}

// Repo returns the goutils Repo
func (m *ModuleExt) Repo() *goutils.Repo {
	m.chkSetGraph("Repo")
	return m.repo
}

func (m *ModuleExt) chkSetGraph(funcName string) {
	if m.repo == nil {
		panic("ERROR: Must call ModuleExt.SetGraph() before calling ModuleExt." + funcName + "()")
	}
}

// IsInFlux checks if this module is in-flux (not ready for release)
// Returns: inFlux bool, reason string, error
// A module is in-flux if:
// - Has in-flux dependencies (pseudo-versions, local replaces)
// - Working tree is dirty (untracked/staged/unstaged files)
// - Has replace directives in go.mod
// - Not tagged or tagged but not pushed (handled separately in engine)
func (m *ModuleExt) IsInFlux(ctx context.Context) (inFlux bool, reason string, err error) {
	var status goutils.Status
	var isDirty bool
	var repo *gitutils.Repo

	// Check dependency status via goutils
	status = m.Module.AnalyzeStatus()
	if status.InFlux {
		inFlux = true
		reason = "has in-flux dependencies"
		goto end
	}

	// Check git dirty state for this specific module (excluding submodules)
	repo, err = gitutils.Open(m.Repo().DirPath)
	if err != nil {
		// If not a git repo, skip this check
		err = nil
	} else {
		var modRelPath dt.PathSegments
		var excludePaths []dt.PathSegments
		var counts gitutils.StatusCounts

		// Get module path relative to repo
		modRelPath, err = m.Dir().Rel(m.Repo().DirPath)
		if err != nil {
			// Can't determine relative path - fall back to whole repo check
			isDirty, err = repo.IsDirty()
			if err != nil {
				goto end
			}
		} else {
			// Get list of submodules to exclude
			excludePaths = m.getSubmodulePathsToExclude()

			// Check status for this module only, excluding submodules
			counts, err = repo.StatusCountsInPathExcluding(modRelPath, excludePaths)
			if err != nil {
				goto end
			}
			isDirty = counts.Staged > 0 || counts.Unstaged > 0 || counts.Untracked > 0
		}

		if isDirty {
			inFlux = true
			reason = "dirty working tree"
			goto end
		}
	}

	// Check for replace directives
	if m.HasReplaceDirectives() {
		inFlux = true
		reason = "has replace directives"
		goto end
	}

end:
	return inFlux, reason, err
}

// getSubmodulePathsToExclude returns paths of other modules in the same repo that should be excluded
func (m *ModuleExt) getSubmodulePathsToExclude() (excludePaths []dt.PathSegments) {
	if m.graph == nil {
		return nil
	}

	// Get all modules in this repo
	repoModules, ok := m.graph.ModulesMapByModulePathByRepoDir[m.Repo().DirPath]
	if !ok {
		return nil
	}

	for otherModExt := range repoModules.Values() {
		// Skip ourselves
		if otherModExt.Dir() == m.Dir() {
			continue
		}

		// Check if this other module is a subdirectory of our module
		relPath, err := otherModExt.Dir().Rel(m.Dir())
		if err != nil {
			// Not a subdirectory, skip
			continue
		}

		// This is a submodule that should be excluded
		excludePaths = append(excludePaths, relPath)
	}

	return excludePaths
}
