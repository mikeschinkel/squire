package retinue

import (
	"errors"
	"maps"
	"slices"

	"github.com/mikeschinkel/go-dt/dtx"
)

type Repo struct {
	DirPath     RepoDir
	Graph       *GoModGraph
	modulesMap  ModuleMapByModulePath
	requireDirs []ModuleDir
}

func NewRepo(repoDir RepoDir) *Repo {
	return &Repo{
		DirPath:    repoDir,
		modulesMap: make(ModuleMapByModulePath),
	}
}

var ErrModulesNotFoundForRepo = errors.New("modules not found for repository")

func (m *Repo) SetGraph(graph *GoModGraph) (err error) {
	var ok bool
	m.Graph = graph
	m.modulesMap, ok = graph.ModulesMapByModulePathByRepoDir[m.DirPath]
	if !ok {
		err = NewErr(ErrModulesNotFoundForRepo,
			"repo_dir", m.DirPath,
		)
	}
	return err
}

func (m *Repo) GoModulePaths() (gms []ModulePath) {
	m.chkSetGraph("GoModulePaths")
	return slices.Collect(maps.Keys(m.modulesMap))
}

func (m *Repo) GoModules() (gms []*GoModule) {
	m.chkSetGraph("GoModules")
	return slices.Collect(maps.Values(m.modulesMap))
}

func (m *Repo) chkSetGraph(funcName string) {
	if len(m.modulesMap) == 0 {
		dtx.Panicf("ERROR: Must call Repo.SetGraph() before calling Repo.%s()", funcName)
	}
}

func (m *Repo) UniqueModulePaths() (unique []ModulePath) {
	gms := m.GoModules()
	uniqueMap := make(map[ModulePath]struct{}, len(gms))
	for _, module := range gms {
		for _, r := range module.Require {
			uniqueMap[ModulePath(r.Mod.Path)] = struct{}{}
		}
	}
	return slices.Collect(maps.Keys(uniqueMap))
}

func (m *Repo) RequireDirs() (requireDirs []ModuleDir) {
	var unique []ModulePath
	var errs []error
	var n int

	if m.requireDirs != nil {
		goto end
	}
	m.chkSetGraph("RequireDirs")
	unique = m.UniqueModulePaths()
	requireDirs = make([]ModuleDir, 0, len(unique))
	for _, mp := range m.UniqueModulePaths() {
		moduleDirs, ok := m.Graph.ModuleDirByModulePath[mp]
		if !ok {
			// Mould must be an import that we are not developing locally, so skip it
			continue
		}
		switch len(moduleDirs) {
		case 0:
			// A module has no local source, we don't need to concern ourselves with it
			continue
		case 1:
			//modDir := moduleDirs.DirPath()
			//repoDir, ok := m.Graph.RepoDirsByModuleDir[modDir]
			//if !ok {
			//	dtx.Panicf("repo not found for Go module %s", modDir)
			//}
			requireDirs = append(requireDirs, moduleDirs.DirPath())
		default:
			errs = AppendErr(errs, NewErr(ErrMultipleGoModulesFound, "files", moduleDirs.DirPaths()))
		}
		n++
	}
	m.requireDirs = requireDirs[:n]
end:
	return m.requireDirs
}
