package retinue

import (
	"fmt"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
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
