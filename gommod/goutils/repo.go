package goutils

import (
	"errors"
	"slices"

	"github.com/mikeschinkel/go-dt/dtx"
)

// Repo represents a git repository containing one or more Go modules
type Repo struct {
	DirPath     RepoDir
	ModuleGraph *ModuleGraph
	modulesMap  ModuleMapByModulePath
	requireDirs []ModuleDir
}

func NewRepo(repoDir RepoDir) *Repo {
	return &Repo{
		DirPath: repoDir,
	}
}

var ErrModulesNotFoundForRepo = errors.New("modules not found for repository")

func (r *Repo) SetGraph(graph *ModuleGraph) (err error) {
	var ok bool
	r.ModuleGraph = graph
	r.modulesMap, ok = graph.ModulesMapByModulePathByRepoDir[r.DirPath]
	if !ok {
		err = NewErr(ErrModulesNotFoundForRepo,
			"repo_dir", r.DirPath,
		)
	}
	return err
}

func (r *Repo) ModulePaths() (mps []ModulePath) {
	r.chkSetGraph("ModulePaths")
	return r.modulesMap.GetKeys()
}

func (r *Repo) Modules() (modules []*Module) {
	r.chkSetGraph("Modules")
	return r.modulesMap.GetValues()
}

func (r *Repo) chkSetGraph(funcName string) {
	if r.modulesMap == nil {
		dtx.Panicf("ERROR: Must call Repo.SetGraph() before calling Repo.%s()", funcName)
	}
}

func (r *Repo) UniqueModulePaths() (unique []ModulePath) {
	modules := r.Modules()
	um := dtx.NewOrderedMap[ModulePath, struct{}](len(modules))
	for _, module := range modules {
		for _, req := range module.Requires {
			um.Set(req.Path, struct{}{})
		}
	}
	return slices.Collect(um.Keys())
}

func (r *Repo) RequireDirs() (requireDirs []ModuleDir) {
	var unique []ModulePath
	var errs []error
	var n int

	if r.requireDirs != nil {
		goto end
	}
	r.chkSetGraph("RequireDirs")
	unique = r.UniqueModulePaths()
	requireDirs = make([]ModuleDir, 0, len(unique))
	for _, mp := range r.UniqueModulePaths() {
		moduleDirs, ok := r.ModuleGraph.ModuleDirByModulePath[mp]
		if !ok {
			// Module must be an import that we are not developing locally, so skip it
			continue
		}
		switch len(moduleDirs) {
		case 0:
			// A module has no local source, we don't need to concern ourselves with it
			continue
		case 1:
			requireDirs = append(requireDirs, moduleDirs.DirPath())
		default:
			errs = AppendErr(errs, NewErr(ErrMultipleGoModulesFound, "files", moduleDirs.DirPaths()))
		}
		n++
	}
	r.requireDirs = requireDirs[:n]
end:

	return r.requireDirs
}
