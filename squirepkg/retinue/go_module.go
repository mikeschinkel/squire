package retinue

import (
	"context"
	"fmt"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/squire/squirepkg/gitutils"
	"github.com/mikeschinkel/squire/squirepkg/modutils"
	"golang.org/x/mod/modfile"
)

type parsedModFile = modfile.File

type ModuleFile = dt.Filepath
type GoModule struct {
	ModuleFile ModuleFile
	Graph      *GoModGraph
	repo       *Repo
	*parsedModFile
	loaded bool
}

func NewGoModule(modFile dt.Filepath) *GoModule {
	return &GoModule{ModuleFile: modFile}
}

func (m *GoModule) SetGraph(graph *GoModGraph) (err error) {
	var ok bool
	m.repo, ok = graph.ReposByModuleDir[m.ModuleFile.Dir()]
	if !ok {
		err = NewErr(ErrRepoNotFoundForGoModule,
			"module_file", m.ModuleFile,
		)
	}
	m.Graph = graph
	return err
}

// Load reads and parses a go.mod file
func (m *GoModule) Load() (err error) {
	var content []byte

	content, err = m.ModuleFile.ReadFile()
	if err != nil {
		goto end
	}

	// Use ParseLax for syntax-only validation (go.mod may not be buildable during local dev)
	m.parsedModFile, err = modfile.ParseLax(string(m.ModuleFile), content, nil)
	if err != nil {
		goto end
	}
	if m.parsedModFile.Module == nil {
		err = NewErr(ErrGoModuleNameNotParsed, "filepath", m.ModuleFile)
		goto end
	}

	m.loaded = true
end:
	return err
}

type ModuleKey string

func (m *GoModule) Key() ModuleKey {
	return ModuleKey(fmt.Sprintf("%s:%s", m.ModulePath(), m.ModuleFile))
}

func (m *GoModule) Dir() dt.DirPath {
	return m.ModuleFile.Dir()
}

func (m *GoModule) chkLoaded(funcName string) {
	if !m.loaded {
		dtx.Panicf("ERROR: Must call GoModule.Load() before GoModule.%s()", funcName)
	}
}

func (m *GoModule) ModulePath() (mp ModulePath) {
	m.chkLoaded("ModulePath")
	return ModulePath(m.parsedModFile.Module.Mod.Path)
}

func (m *GoModule) Requires() (mps []ModulePath) {
	mps = make([]ModulePath, len(m.parsedModFile.Require))
	for i, require := range m.parsedModFile.Require {
		mps[i] = ModulePath(require.Mod.Path)
	}
	return mps
}

func (m *GoModule) RequiredModulePaths() (names []ModulePath) {
	names = make([]ModulePath, len(m.parsedModFile.Require))
	for i, r := range m.parsedModFile.Require {
		names[i] = ModulePath(r.Mod.Path)
	}
	return names
}

func (m *GoModule) Repo() (repo *Repo) {
	m.chkSetGraph("Repo")
	return m.repo
}

func (m *GoModule) chkSetGraph(funcName string) {
	if m.repo == nil {
		dtx.Panicf("ERROR: Must call GoModule.SetGraph() before calling GoModule.%s()", funcName)
	}
}

// RequireDirs returns the module directories that this specific module depends on
func (m *GoModule) RequireDirs() (requireDirs []ModuleDir) {
	m.chkLoaded("RequireDirs")
	m.chkSetGraph("RequireDirs")

	requireDirs = make([]ModuleDir, 0, len(m.Require))
	for _, require := range m.Require {
		mp := ModulePath(require.Mod.Path)
		moduleDirs, ok := m.Graph.ModuleDirByModulePath[mp]
		if !ok {
			// Not a local module, skip it
			continue
		}
		switch len(moduleDirs) {
		case 0:
			continue
		case 1:
			requireDirs = append(requireDirs, moduleDirs.DirPath())
		default:
			// Multiple modules with same path - this shouldn't happen in practice
			// but handle it by taking the first one
			requireDirs = append(requireDirs, moduleDirs.DirPath())
		}
	}
	return requireDirs // DETERMINISTIIC ORDER? CHECK!
}

// AnalyzeStatus returns the dependency status for this module's go.mod
func (m *GoModule) AnalyzeStatus() (status modutils.Status, err error) {
	m.chkLoaded("AnalyzeStatus")

	// TODO We already have a load3ed retinue.GoModule, not need Load a modutils.Module.
	mod := modutils.NewModule(m.ModuleFile)
	err = mod.Load()
	if err != nil {
		goto end
	}

	// TODO We should move AnalyzeStatus to retinue.GoModule
	status = modutils.AnalyzeStatus(mod)

end:
	return status, err
}

// IsInFlux checks if this module is in-flux (not ready for release)
// Returns: inFlux bool, reason string, error
// A module is in-flux if:
// - Has in-flux dependencies (pseudo-versions, local replaces)
// - Working tree is dirty (untracked/staged/unstaged files)
// - Has replace directives in go.mod
// - Not tagged or tagged but not pushed (handled separately in engine)
func (m *GoModule) IsInFlux(ctx context.Context) (inFlux bool, reason string, err error) {
	m.chkLoaded("IsInFlux")
	m.chkSetGraph("IsInFlux")

	var status modutils.Status
	var isDirty bool
	var repo *gitutils.Repo

	// Check dependency status via modutils
	status, err = m.AnalyzeStatus()
	if err != nil {
		goto end
	}
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
	if len(m.parsedModFile.Replace) > 0 {
		inFlux = true
		reason = "has replace directives"
		goto end
	}

end:
	return inFlux, reason, err
}

// getSubmodulePathsToExclude returns paths of other modules in the same repo that should be excluded
func (m *GoModule) getSubmodulePathsToExclude() (excludePaths []dt.PathSegments) {
	if m.Graph == nil {
		return nil
	}

	// Get all modules in this repo
	repoModules, ok := m.Graph.ModulesMapByModulePathByRepoDir[m.Repo().DirPath]
	if !ok {
		return nil
	}

	for otherMod := range repoModules.Values() {
		// Skip ourselves
		if otherMod.Dir() == m.Dir() {
			continue
		}

		// Check if this other module is a subdirectory of our module
		relPath, err := otherMod.Dir().Rel(m.Dir())
		if err != nil {
			// Not a subdirectory, skip
			continue
		}

		// This is a submodule that should be excluded
		excludePaths = append(excludePaths, relPath)
	}

	return excludePaths
}

// HasReplaceDirectives checks if this module's go.mod has any replace directives
func (m *GoModule) HasReplaceDirectives() bool {
	m.chkLoaded("HasReplaceDirectives")
	return len(m.parsedModFile.Replace) > 0
}
