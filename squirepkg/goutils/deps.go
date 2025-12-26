package goutils

import (
	"fmt"

	"github.com/mikeschinkel/go-dt"
	"golang.org/x/mod/module"
)

type DependencyState struct {
	Path    ModulePath
	Version dt.Version
	Replace string
	InFlux  bool
	Reason  string
}

type Status struct {
	Deps   []DependencyState
	InFlux bool
}

type replacesMap map[string]Replace

// AnalyzeStatus analyzes the dependency state of this module
func (m *Module) AnalyzeStatus() (status Status) {
	m.chkLoaded("AnalyzeStatus")

	replaces := make(replacesMap)
	for _, rep := range m.Replaces {
		key := fmt.Sprintf("%s@%s", rep.Old.Path, rep.Old.Version)
		replaces[key] = rep
	}

	var deps []DependencyState
	inFlux := false

	for _, req := range m.Requires {
		dep, skip := analyzeRequire(req, replaces)
		if skip {
			continue
		}
		if dep.InFlux {
			inFlux = true
		}
		deps = append(deps, dep)
	}

	return Status{
		Deps:   deps,
		InFlux: inFlux,
	}
}

func analyzeRequire(req Require, replaces replacesMap) (ds DependencyState, skip bool) {
	var rep Replace
	var ok bool

	if req.Indirect {
		skip = true
		goto end
	}

	ds = DependencyState{
		Path:    req.Path,
		Version: req.Version,
	}

	if module.IsPseudoVersion(string(req.Version)) {
		ds.InFlux = true
		ds.Reason = "unreleased VCS pseudo-version"
	}

	rep, ok = replaces[req.PathAtVersion()]
	if !ok {
		rep, ok = replaces[req.PathAt()]
	}
	if ok {
		switch {
		case rep.New.Version == "" && rep.New.Path.maybeLocalPath():
			ds.InFlux = true
			ds.Reason = "local replace"
		case rep.New.Version != "" && module.IsPseudoVersion(string(rep.New.Version)):
			ds.InFlux = true
			ds.Reason = "replace to unreleased VCS pseudo-version"
		}
		ds.Replace = rep.New.PathAtVersion()
	}

end:
	return ds, skip
}
