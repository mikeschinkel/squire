package goutils

import (
	"fmt"
	"strings"

	"github.com/mikeschinkel/go-dt"
	"golang.org/x/mod/modfile"
)

type ModulePath string

func (mp ModulePath) LooksPublishable() bool {
	// TODO Improve this logic
	first, _, _ := strings.Cut(string(mp), "/")
	return strings.Contains(first, ".")
}
func (mp ModulePath) maybeLocalPath() (looksLocal bool) {
	var dp dt.DirPath
	if mp == "" {
		goto end
	}
	looksLocal = true
	dp = dt.DirPath(mp)
	switch {
	case dp.IsTidlePath():
	case dp.IsAbs():
	default:
		looksLocal = false
	}
end:
	return looksLocal
}

type Module struct {
	// Core fields (from original goutils)
	Path     ModulePath  // Module path from go.mod
	Filepath dt.Filepath // Path to go.mod file
	Requires []Require   // Required dependencies
	Replaces []Replace   // Replace directives

	// Graph integration (optional - from squiresvc)
	Graph *ModuleGraph // Optional: dependency graph (nil if standalone)
	repo  *Repo        // Optional: parent repository (nil if standalone)

	// Parsed structure (from squiresvc)
	modfile *modfile.File // Parsed go.mod structure
	loaded  bool          // Whether Load() has been called
}

func NewModule(goModPath dt.Filepath) *Module {
	return &Module{
		Filepath: goModPath,
		Requires: make([]Require, 0),
		Replaces: make([]Replace, 0),
		loaded:   false,
		Graph:    nil,
		repo:     nil,
	}
}

func (m *Module) LooksPublishable() bool {
	return m.Path.LooksPublishable()
}

func (m *Module) Load() (err error) {
	var content []byte
	var parsed *modfile.File

	content, err = m.Filepath.ReadFile()
	if err != nil {
		goto end
	}

	// Parse (syntax-only validation - intentional for local dev)
	parsed, err = modfile.ParseLax(string(m.Filepath), content, nil)
	if err != nil {
		goto end
	}

	if parsed.Module == nil {
		err = fmt.Errorf("missing module directive")
		goto end
	}

	// Store parsed structure
	m.modfile = parsed

	// Extract module path
	m.Path = ModulePath(parsed.Module.Mod.Path)

	// Extract requires
	m.Requires = make([]Require, 0, len(parsed.Require))
	for _, req := range parsed.Require {
		m.Requires = append(m.Requires, NewRequire(
			NewPathVersion(ModulePath(req.Mod.Path), dt.Version(req.Mod.Version)),
			req.Indirect,
		))
	}

	// Extract replaces
	m.Replaces = make([]Replace, 0, len(parsed.Replace))
	for _, rep := range parsed.Replace {
		m.Replaces = append(m.Replaces, NewReplace(
			NewPathVersion(ModulePath(rep.Old.Path), dt.Version(rep.Old.Version)),
			NewPathVersion(ModulePath(rep.New.Path), dt.Version(rep.New.Version)),
		))
	}

	m.loaded = true

end:
	return err
}

// SetGraph associates a ModuleGraph with this Module (required for graph-aware methods)
func (m *Module) SetGraph(graph *ModuleGraph) error {
	m.chkLoaded("SetGraph")
	m.Graph = graph
	return nil
}

// Dir returns the directory containing the go.mod file
func (m *Module) Dir() dt.DirPath {
	m.chkLoaded("Dir")
	return m.Filepath.Dir()
}

// Key returns a unique key for this module (module path)
func (m *Module) Key() ModuleKey {
	m.chkLoaded("Key")
	return ModuleKey(m.Path)
}

// Repo returns the repository containing this module (may be nil)
func (m *Module) Repo() *Repo {
	m.chkSetGraph("Repo")
	return m.repo
}

// RequiredModulePaths returns paths of all direct dependencies
func (m *Module) RequiredModulePaths() []ModulePath {
	m.chkLoaded("RequiredModulePaths")
	paths := make([]ModulePath, 0, len(m.Requires))
	for _, req := range m.Requires {
		if !req.Indirect {
			paths = append(paths, req.Path)
		}
	}
	return paths
}

// RequireDirs returns directories of required modules (graph-aware)
func (m *Module) RequireDirs() []ModuleDir {
	m.chkSetGraph("RequireDirs")
	var dirs []ModuleDir
	for _, req := range m.Requires {
		if !req.Indirect {
			if mod, ok := m.Graph.modules[ModuleKey(req.Path)]; ok {
				dirs = append(dirs, ModuleDir(mod.Dir()))
			}
		}
	}
	return dirs
}

// HasReplaceDirectives returns true if module has any replace directives
func (m *Module) HasReplaceDirectives() bool {
	m.chkLoaded("HasReplaceDirectives")
	return len(m.Replaces) > 0
}

// chkLoaded panics if Load() hasn't been called yet
func (m *Module) chkLoaded(funcName string) {
	if !m.loaded {
		panic(fmt.Sprintf("%s() called before Load()", funcName))
	}
}

// chkSetGraph panics if SetGraph() hasn't been called yet
func (m *Module) chkSetGraph(funcName string) {
	m.chkLoaded(funcName)
	if m.Graph == nil {
		panic(fmt.Sprintf("%s() requires SetGraph() to be called first", funcName))
	}
}
