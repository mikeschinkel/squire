package modutils

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
	Path     ModulePath
	Filepath dt.Filepath
	Requires []Require
	Replaces []Replace
}

func NewModule(goModPath dt.Filepath) Module {
	return Module{
		Filepath: goModPath,
		Requires: make([]Require, 0),
		Replaces: make([]Replace, 0),
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

	parsed, err = modfile.Parse(string(m.Path), content, nil)
	if err != nil {
		goto end
	}

	if parsed.Module == nil {
		err = fmt.Errorf("missing module directive")
		goto end
	}

	m.Path = ModulePath(parsed.Module.Mod.Path)

	for _, req := range parsed.Require {
		m.Requires = append(m.Requires, NewRequire(
			NewPathVersion(ModulePath(req.Mod.Path), dt.Version(req.Mod.Version)),
			req.Indirect,
		))
	}
	for _, rep := range parsed.Replace {
		m.Replaces = append(m.Replaces, NewReplace(
			NewPathVersion(ModulePath(rep.Old.Path), dt.Version(rep.Old.Version)),
			NewPathVersion(ModulePath(rep.New.Path), dt.Version(rep.New.Version)),
		))
	}
end:
	return err
}
