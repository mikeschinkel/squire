package goutils

import (
	"sort"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
)

// Type aliases for clarity and future flexibility

// ModuleKey uniquely identifies a module (typically the module path)
type ModuleKey string

// ModuleDir represents a module's directory path
type ModuleDir = dt.DirPath

// RepoDir represents a repository's directory path
type RepoDir = dt.DirPath

// ModuleDirMap maps module directories (for handling duplicate module paths)
type ModuleDirMap map[ModuleDir]struct{}

func (m ModuleDirMap) DirPaths() (dps []dt.DirPath) {
	dps = make([]dt.DirPath, 0, len(m))
	for dp := range m {
		dps = append(dps, dp)
	}
	// Sort for deterministic ordering
	sort.Slice(dps, func(i, j int) bool {
		return string(dps[i]) < string(dps[j])
	})
	return dps
}

func (m ModuleDirMap) DirPath() (dp dt.DirPath) {
	// Get sorted paths for deterministic ordering
	paths := m.DirPaths()
	if len(paths) > 0 {
		dp = paths[0]
	}
	// Debug: log when multiple paths exist for same module
	if len(paths) > 1 {
		// This shouldn't happen in practice - log if it does
		_ = paths // suppress unused warning
	}
	return dp
}

// ModuleMapByModulePath is an ordered map of modules by module path
type ModuleMapByModulePath = *dtx.OrderedMap[ModulePath, *Module]

// ModulesMapByModulePathByRepoDir maps repo directories to their module maps
type ModulesMapByModulePathByRepoDir = map[RepoDir]ModuleMapByModulePath

// RepoDirsByModuleDir maps module directories to their repo directories
type RepoDirsByModuleDir = *dtx.OrderedMap[ModuleDir, RepoDir]
