package retinue

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"

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

type ModuleMapByModulePath map[ModulePath]*GoModule

func (mm ModuleMapByModulePath) Requires(g *GoModGraph) (requires []ModulePath) {
	unique := make(map[ModulePath]struct{}, len(mm))
	for _, module := range mm {
		for _, r := range module.Require {
			unique[ModulePath(r.Mod.Path)] = struct{}{}
		}
	}
	requires = make([]ModulePath, len(unique))
	n := 0
	for _, mp := range slices.Collect(maps.Keys(unique)) {
		_, ok := g.ModuleDirByModulePath[mp]
		if !ok {
			// No local repo found
			continue
		}
		requires[n] = mp
		n++
	}
	return requires[:n]
}

type RepoDir = dt.DirPath
type ModuleDir = dt.DirPath
type RepoDirsByModuleDir map[ModuleDir]RepoDir
type GoModGraph struct {
	RepoDir                         dt.DirPath
	GoModFiles                      []dt.Filepath
	ModulesMapByModulePathByRepoDir map[RepoDir]ModuleMapByModulePath
	ModulesByKey                    map[ModuleKey]*GoModule
	ModulesByModuleDir              map[ModuleDir]*GoModule
	ModulePathsByModuleDir          map[ModuleDir]ModulePath
	ModuleDirByModulePath           map[ModulePath]ModuleDirMap
	RepoDirsByModuleDir             RepoDirsByModuleDir
	ReposByRepoDir                  map[RepoDir]*Repo
	ReposByModuleDir                map[ModuleDir]*Repo
	moduleDirVisited                map[dt.DirPath]struct{}

	Writer cliutil.Writer
	Logger *slog.Logger

	//ReposByModulePath    map[ModulePath]RepoDirsByModuleDir
	//Requires             []ModulePath
	//RequiredBy            map[ModulePath]ModulePathMap
}
type GoModuleGraphArgs struct {
	Writer cliutil.Writer
	Logger *slog.Logger
}

func NewGoModuleGraph(repoDir dt.DirPath, files []dt.Filepath, args GoModuleGraphArgs) *GoModGraph {
	return &GoModGraph{
		RepoDir:                         repoDir,
		GoModFiles:                      files,
		ModulesMapByModulePathByRepoDir: make(map[RepoDir]ModuleMapByModulePath),
		ModulesByKey:                    make(map[ModuleKey]*GoModule),
		ModulesByModuleDir:              make(map[ModuleDir]*GoModule),
		ModulePathsByModuleDir:          make(map[ModuleDir]ModulePath),
		ModuleDirByModulePath:           make(map[ModulePath]ModuleDirMap),
		RepoDirsByModuleDir:             make(map[ModuleDir]RepoDir),
		ReposByRepoDir:                  make(map[RepoDir]*Repo),
		ReposByModuleDir:                make(map[ModuleDir]*Repo),
		//ReposByModulePath:    make(map[ModulePath]RepoDirsByModuleDir),
		//Requires:             make([]ModulePath, 0),

		// moduleDirVisited is a cache of visits we we don't repeatedly visit the same modules
		moduleDirVisited: make(map[dt.DirPath]struct{}),
		Writer:           args.Writer,
		Logger:           args.Logger,
	}
}

var ErrNoGoModuleFound = errors.New("no Go modules found")

//goland:noinspection GoErrorStringFormat
var ErrMultipleGoModulesFound = errors.New("multiple Go modules found")

func (g *GoModGraph) Traverse() (results []string, err error) {

	// Get the modules required for this repo
	repo, ok := g.ReposByRepoDir[g.RepoDir]
	if !ok {
		err = NewErr(ErrNoGoModuleFound, "repo", g.RepoDir)
		goto end
	}
	// Now traverse the unique requires for those modules that have local repos
	results, err = g.traverseModule(repo.RequireDirs())
end:
	return results, err
}

func (g *GoModGraph) traverseModule(modDirs []ModuleDir) (results []string, err error) {
	var errs []error
	for _, modDir := range modDirs {
		_, ok := g.moduleDirVisited[modDir]
		if ok {
			// Already processed this module, skip it
			continue
		}
		g.moduleDirVisited[modDir] = dtx.NULL{}
		var repo *Repo
		repo, ok = g.ReposByModuleDir[modDir]
		if !ok {
			dtx.Panicf("repo not found for Go module %s", modDir)
		}
		err = repo.SetGraph(g)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		var these []string
		these, err = g.traverseModule(repo.RequireDirs())
		if err != nil {
			errs = append(errs, err)
			continue
		}
		these = append(these, fmt.Sprintf("Repo: %s, Module %s", repo.DirPath, modDir))
		results = append(results, these...)
	}
	return results, err
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

		var repoMods ModuleMapByModulePath
		repoMods, ok = g.ModulesMapByModulePathByRepoDir[repoDir]
		if !ok {
			repoMods = make(map[ModulePath]*GoModule)
			g.ModulesMapByModulePathByRepoDir[repoDir] = repoMods
		}
		_, ok = repoMods[gm.ModulePath()]
		if !ok {
			repoMods[gm.ModulePath()] = gm
		}

		//for _, module := range repoMods {
		//	mp := module.ModulePath()
		//	modDir := module.Dir()
		//	mpRepos, ok := g.ReposByModulePath[mp]
		//	if !ok {
		//		_, ok := mpRepos[modDir]
		//		if !ok {
		//			mpRepos = make(RepoDirsByModuleDir)
		//		}
		//	}
		//	mpRepos[modDir] = repoDir
		//	g.ReposByModulePath[mp] = mpRepos
		//}

		for _, module := range repoMods {
			g.RepoDirsByModuleDir[module.Dir()] = repoDir
		}

		key := gm.Key()
		mp := gm.ModulePath()
		_, ok = g.ModulesByKey[key]
		if !ok {
			g.ModulesByKey[key] = gm
		}
		_, ok = g.ModulesByModuleDir[modDir]
		if !ok {
			g.ModulesByModuleDir[modDir] = gm
		}
		_, ok = g.ModulePathsByModuleDir[modDir]
		if !ok {
			g.ModulePathsByModuleDir[modDir] = mp
		}
		var dpMap ModuleDirMap
		dpMap, ok = g.ModuleDirByModulePath[mp]
		if !ok {
			dpMap = ModuleDirMap{}
			g.ModuleDirByModulePath[mp] = dpMap
		}
		dpMap[modDir] = struct{}{}
		//g.Requires = gm.RequiredModulePaths()
		//for _, dep := range g.Requires {
		//	_, ok = g.ModuleDirByModulePath[dep]
		//	if !ok {
		//		g.ModuleDirByModulePath[dep] = make(ModuleDirMap)
		//	}
		//}
	}
	for mp, dp := range g.ModuleDirByModulePath {
		if len(dp) > 0 {
			continue
		}
		delete(g.ModuleDirByModulePath, mp)
	}

	for modDir, repoDir := range g.RepoDirsByModuleDir {
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
