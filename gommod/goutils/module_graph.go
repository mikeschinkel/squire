package goutils

import (
	"errors"
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
)

// ModuleMapRequires returns the unique module paths required by modules in the map
func ModuleMapRequires(mm ModuleMapByModulePath, g *ModuleGraph) (requires []ModulePath) {
	unique := make(map[ModulePath]struct{})

	// Iterate using OrderedMap's Values() iterator
	for module := range mm.Values() {
		for _, r := range module.Requires {
			unique[r.Path] = struct{}{}
		}
	}

	requires = make([]ModulePath, 0, len(unique))
	for mp := range unique {
		_, ok := g.ModuleDirByModulePath[mp]
		if !ok {
			// No local repo found
			continue
		}
		requires = append(requires, mp)
	}
	return requires
}

// ModuleGraph represents a dependency graph of Go modules
type ModuleGraph struct {
	RepoDir    dt.DirPath
	GoModFiles []dt.Filepath

	// Use OrderedMap for deterministic iteration
	ModulesMapByModulePathByRepoDir ModulesMapByModulePathByRepoDir
	RepoDirsByModuleDir             RepoDirsByModuleDir

	// Regular maps for lookups only
	modules               map[ModuleKey]*Module
	ModulesByModuleDir    map[ModuleDir]*Module
	ModuleDirByModulePath map[ModulePath]ModuleDirMap
	ReposByRepoDir        map[RepoDir]*Repo
	ReposByModuleDir      map[ModuleDir]*Repo
	moduleDirVisited      map[dt.DirPath]struct{}

	Writer cliutil.Writer
	Logger *slog.Logger
}

type ModuleGraphArgs struct {
	Writer cliutil.Writer
	Logger *slog.Logger
}

func NewGraph(repoDir dt.DirPath, files []dt.Filepath, args ModuleGraphArgs) *ModuleGraph {
	return &ModuleGraph{
		RepoDir:                         repoDir,
		GoModFiles:                      files,
		ModulesMapByModulePathByRepoDir: make(ModulesMapByModulePathByRepoDir),
		RepoDirsByModuleDir:             dtx.NewOrderedMap[ModuleDir, RepoDir](len(files)),
		modules:                         make(map[ModuleKey]*Module),
		ModulesByModuleDir:              make(map[ModuleDir]*Module),
		ModuleDirByModulePath:           make(map[ModulePath]ModuleDirMap),
		ReposByRepoDir:                  make(map[RepoDir]*Repo),
		ReposByModuleDir:                make(map[ModuleDir]*Repo),

		// moduleDirVisited is a cache of visits so we don't repeatedly visit the same modules
		moduleDirVisited: make(map[dt.DirPath]struct{}),
		Writer:           args.Writer,
		Logger:           args.Logger,
	}
}

var ErrNoGoModuleFound = errors.New("no Go modules found")

//goland:noinspection GoErrorStringFormat
var ErrMultipleGoModulesFound = errors.New("multiple Go modules found")

type TraverseResult struct {
	RepoModules *dtx.OrderedMap[RepoDir, []ModuleDir]
}

func (g *ModuleGraph) Traverse() (result *TraverseResult, err error) {
	result = &TraverseResult{
		RepoModules: dtx.NewOrderedMap[RepoDir, []ModuleDir](10),
	}

	// Get the modules required for this repo
	repo, ok := g.ReposByRepoDir[g.RepoDir]
	if !ok {
		err = NewErr(ErrNoGoModuleFound, "repo", g.RepoDir)
		goto end
	}

	// Now traverse the unique requires for those modules that have local repos
	err = g.traverseModule(repo.RequireDirs(), result)
end:
	return result, err
}

func (g *ModuleGraph) traverseModule(modDirs []ModuleDir, result *TraverseResult) (err error) {
	var errs []error

	for _, modDir := range modDirs {
		_, ok := g.moduleDirVisited[modDir]
		if ok {
			// Already processed this module, skip it
			continue
		}
		g.moduleDirVisited[modDir] = dtx.NULL{}

		// Get the specific module at this directory
		module, ok := g.ModulesByModuleDir[modDir]
		if !ok {
			dtx.Panicf("module not found for directory %s", modDir)
		}

		// Ensure module has graph set (needed for RequireDirs)
		err = module.SetGraph(g)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Recursively process dependencies for THIS SPECIFIC MODULE first
		err = g.traverseModule(module.RequireDirs(), result)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Get the repo for this module
		repo, ok := g.ReposByModuleDir[modDir]
		if !ok {
			dtx.Panicf("repo not found for Go module %s", modDir)
		}

		// Add this module to the repo's module list in the OrderedMap
		modules, exists := result.RepoModules.Get(repo.DirPath)
		if !exists {
			modules = []ModuleDir{}
		}
		modules = append(modules, modDir)
		result.RepoModules.Set(repo.DirPath, modules)
	}

	return CombineErrs(errs)
}

func (g *ModuleGraph) Build() (err error) {
	var errs []error

	for _, modFile := range g.GoModFiles {
		var ok bool
		module := NewModule(modFile)
		err = module.Load()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		modDir := modFile.Dir()
		var repoDir RepoDir
		repoDir, err = FindRepoRoot(modDir)
		if err != nil {
			g.Writer.Errorf("Git repository not found for %s\n", modDir)
			continue
		}

		// Get or create OrderedMap for this repo
		var repoMods ModuleMapByModulePath
		repoMods, ok = g.ModulesMapByModulePathByRepoDir[repoDir]
		if !ok {
			repoMods = dtx.NewOrderedMap[ModulePath, *Module](10)
			g.ModulesMapByModulePathByRepoDir[repoDir] = repoMods
		}

		// Add module if not already present
		_, ok = repoMods.Get(module.Path)
		if !ok {
			repoMods.Set(module.Path, module)
		}

		// Update RepoDirsByModuleDir for all modules in repo
		for mod := range repoMods.Values() {
			g.RepoDirsByModuleDir.Set(mod.Dir(), repoDir)
		}

		// Regular map updates (lookups only, no iteration)
		mp := module.Path
		g.ModulesByModuleDir[modDir] = module
		g.modules[ModuleKey(mp)] = module

		// Update ModuleDirByModulePath
		dpMap, ok := g.ModuleDirByModulePath[mp]
		if !ok {
			dpMap = ModuleDirMap{}
			g.ModuleDirByModulePath[mp] = dpMap
		}
		dpMap[modDir] = struct{}{}
	}

	// Clean up empty entries
	for mp, dp := range g.ModuleDirByModulePath {
		if len(dp) == 0 {
			delete(g.ModuleDirByModulePath, mp)
		}
	}

	// Create Repo objects (now deterministic due to OrderedMap)
	for modDir, repoDir := range g.RepoDirsByModuleDir.Iterator() {
		repo := NewRepo(repoDir)
		err = repo.SetGraph(g)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		g.ReposByModuleDir[modDir] = repo
		g.ReposByRepoDir[repoDir] = repo
	}

	return CombineErrs(errs)
}
