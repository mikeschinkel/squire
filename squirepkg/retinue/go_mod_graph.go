package retinue

import (
	"errors"
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
)

type ModuleDirMap map[ModuleDir]struct{}

func (m ModuleDirMap) DirPaths() (dps []dt.DirPath) {
	dps = make([]dt.DirPath, len(m))
	for dp := range m {
		dps = append(dps, dp)
	}
	return dps
}
func (m ModuleDirMap) DirPath() (dp dt.DirPath) {
	for dp = range m {
		break
	}
	return dp
}

type ModuleMapByModulePath = *dtx.OrderedMap[ModulePath, *GoModule]

// ModuleMapRequires returns the unique module paths required by modules in the map
func ModuleMapRequires(mm ModuleMapByModulePath, g *GoModGraph) (requires []ModulePath) {
	unique := make(map[ModulePath]struct{})

	// Iterate using OrderedMap's Values() iterator
	for module := range mm.Values() {
		for _, r := range module.Require {
			unique[ModulePath(r.Mod.Path)] = struct{}{}
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

type RepoDir = dt.DirPath
type ModuleDir = dt.DirPath
type ModulesMapByModulePathByRepoDir = map[RepoDir]ModuleMapByModulePath
type RepoDirsByModuleDir = *dtx.OrderedMap[ModuleDir, RepoDir]
type GoModGraph struct {
	RepoDir    dt.DirPath
	GoModFiles []dt.Filepath

	// Use OrderedMap for deterministic iteration
	ModulesMapByModulePathByRepoDir ModulesMapByModulePathByRepoDir
	RepoDirsByModuleDir             RepoDirsByModuleDir

	// Regular maps for lookups only
	ModulesByModuleDir    map[ModuleDir]*GoModule
	ModuleDirByModulePath map[ModulePath]ModuleDirMap
	ReposByRepoDir        map[RepoDir]*Repo
	ReposByModuleDir      map[ModuleDir]*Repo
	moduleDirVisited      map[dt.DirPath]struct{}

	Writer cliutil.Writer
	Logger *slog.Logger

	// DELETED: ModulesByKey, ModulePathsByModuleDir (only used for duplicate detection)
}
type GoModuleGraphArgs struct {
	Writer cliutil.Writer
	Logger *slog.Logger
}

func NewGoModuleGraph(repoDir dt.DirPath, files []dt.Filepath, args GoModuleGraphArgs) *GoModGraph {
	return &GoModGraph{
		RepoDir:                         repoDir,
		GoModFiles:                      files,
		ModulesMapByModulePathByRepoDir: make(ModulesMapByModulePathByRepoDir),
		RepoDirsByModuleDir:             dtx.NewOrderedMap[ModuleDir, RepoDir](len(files)),
		ModulesByModuleDir:              make(map[ModuleDir]*GoModule),
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

func (g *GoModGraph) Traverse() (result *TraverseResult, err error) {
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

func (g *GoModGraph) traverseModule(modDirs []ModuleDir, result *TraverseResult) (err error) {
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

func (g *GoModGraph) Build() (err error) {
	var errs []error

	for _, modFile := range g.GoModFiles {
		var ok bool
		gm := NewGoModule(modFile)
		err = gm.Load()
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
			repoMods = dtx.NewOrderedMap[ModulePath, *GoModule](10)
			g.ModulesMapByModulePathByRepoDir[repoDir] = repoMods
		}

		// Add module if not already present
		_, ok = repoMods.Get(gm.ModulePath())
		if !ok {
			repoMods.Set(gm.ModulePath(), gm)
		}

		// Update RepoDirsByModuleDir for all modules in repo
		for module := range repoMods.Values() {
			g.RepoDirsByModuleDir.Set(module.Dir(), repoDir)
		}

		// Regular map updates (lookups only, no iteration)
		mp := gm.ModulePath()
		g.ModulesByModuleDir[modDir] = gm

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
