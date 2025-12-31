package gompkg

import (
	"fmt"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gomion"
	"golang.org/x/mod/modfile"
)

// ModuleKind classifies the role of a module in Gomion's universe.
type ModuleKind int

const (
	UnspecifiedModuleKind ModuleKind = iota
	LibModuleKind
	ExeModuleKind
	TestModuleKind
)

// String returns the string representation of ModuleKind
func (k ModuleKind) String() string {
	switch k {
	case LibModuleKind:
		return "lib"
	case ExeModuleKind:
		return "exe"
	case TestModuleKind:
		return "test"
	default:
		return "unspecified"
	}
}

// Module represents a single Go module that Gomion knows about.
// It includes its location, semantic kind, and dependencies on other modules.
type Module struct {
	// RepoRoot is the filesystem root for the repo containing this module.
	RepoRoot dt.DirPath

	// RelDir is the module's path relative to RepoRoot, e.g. "./", "./cmd", "./test".
	RelDir dt.PathSegments

	// ModulePath is the Go module path from this module's go.mod "module" directive,
	// e.g. "github.com/mikeschinkel/go-dt".
	ModulePath ModulePath

	// Kind classifies the module as lib/cmd/test.
	Kind ModuleKind

	// Versioned indicates whether this module is considered a versionable unit.
	// Phase 2 computes this via heuristics (see determineVersioned).
	Versioned bool

	// Requires lists the Go module paths of Gomion-managed modules that
	// this module depends on (from its go.mod "require" directives).
	//
	// Only requires that are themselves known Gomion-managed modules are included.
	Requires []ModulePath
}

// RepoName returns the human-readable repository name (last component of repo root path)
// Example: ~/Projects/go-pkgs/go-dt -> "go-dt"
func (m *Module) RepoName() (name string) {
	name = string(m.RepoRoot.Base())
	return name
}

// ShortName returns the short module name (last component of module path)
// Example: github.com/mikeschinkel/go-dt -> "go-dt"
func (m *Module) ShortName() (name string) {
	var parts []dt.URLSegment

	// Split module path by /
	parts = m.ModulePath.Split("/")

	// Return last component
	if len(parts) > 0 {
		name = string(parts[len(parts)-1])
		goto end
	}

	// Fallback to full module path if no / found
	name = string(m.ModulePath)

end:
	return name
}

// ModuleSet represents a collection of Modules Gomion knows about,
// along with internal indexes for dependency reasoning.
type ModuleSet struct {
	Modules []*Module

	// byPath provides fast lookup by ModulePath
	byPath map[ModulePath]*Module
}

// NewModuleSet creates a new ModuleSet
func NewModuleSet() *ModuleSet {
	return &ModuleSet{
		Modules: make([]*Module, 0),
		byPath:  make(map[ModulePath]*Module),
	}
}

// Add adds a module to the set
func (ms *ModuleSet) Add(m *Module) {
	ms.Modules = append(ms.Modules, m)
	ms.byPath[m.ModulePath] = m
}

// Get returns a module by its module path
func (ms *ModuleSet) Get(modulePath ModulePath) (*Module, bool) {
	m, ok := ms.byPath[modulePath]
	return m, ok
}

// OrderModules returns all modules in the set in a dependency-safe order.
//
// For any module M in the returned slice, all modules whose ModulePath is in
// M.Requires appear earlier in the slice.
func (ms *ModuleSet) OrderModules() (ordered []*Module, err error) {
	var remaining map[ModulePath]*Module
	var inDegree map[ModulePath]int
	var modulePath ModulePath
	var module *Module
	var depPath ModulePath
	var ready []*Module
	var current *Module
	var dependent *Module

	// Build working map and in-degree counts
	remaining = make(map[ModulePath]*Module)
	inDegree = make(map[ModulePath]int)

	for _, module = range ms.Modules {
		modulePath = module.ModulePath
		remaining[modulePath] = module
		inDegree[modulePath] = 0
	}

	// Count dependencies
	for _, module = range ms.Modules {
		for _, depPath = range module.Requires {
			// Only count dependencies that are in our module set
			if _, ok := remaining[depPath]; ok {
				inDegree[module.ModulePath]++
			}
		}
	}

	// Find modules with no dependencies
	ready = make([]*Module, 0)
	for modulePath, module = range remaining {
		if inDegree[modulePath] == 0 {
			ready = append(ready, module)
		}
	}

	// Process modules in dependency order
	ordered = make([]*Module, 0, len(ms.Modules))

	for len(ready) > 0 {
		// Take the first ready module
		current = ready[0]
		ready = ready[1:]

		// Add to ordered list
		ordered = append(ordered, current)

		// Remove from remaining
		delete(remaining, current.ModulePath)

		// Decrease in-degree for dependents
		for _, dependent = range ms.Modules {
			// Check if dependent requires current
			for _, depPath = range dependent.Requires {
				if depPath == current.ModulePath {
					inDegree[dependent.ModulePath]--
					// If dependent now has no unresolved dependencies, add to ready
					if inDegree[dependent.ModulePath] == 0 {
						if _, stillRemaining := remaining[dependent.ModulePath]; stillRemaining {
							ready = append(ready, dependent)
						}
					}
					break
				}
			}
		}
	}

	// Check for cycles
	if len(remaining) > 0 {
		var cycle []ModulePath
		for modulePath = range remaining {
			cycle = append(cycle, modulePath)
		}
		err = fmt.Errorf("dependency cycle detected among modules: %s", ModulePaths(cycle).Join(", "))
		ordered = nil
		goto end
	}

end:
	return ordered, err
}

// determineModuleKind determines the ModuleKind based on the relative directory
func determineModuleKind(relDir dt.PathSegments) ModuleKind {
	var normalized dt.PathSegments

	normalized = relDir.TrimPrefix("./").ToLower()

	switch normalized {
	case "test", "tests":
		return TestModuleKind
	case "cmd":
		return ExeModuleKind
	default:
		return UnspecifiedModuleKind
	}
	switch {
	// Test modules
	case normalized.HasPrefix("test/"):
		return TestModuleKind
	case normalized.HasPrefix("tests/"):
		return TestModuleKind
	case normalized.HasPrefix("cmd/"):
		return ExeModuleKind

	case normalized.HasPrefix("cmd/"):
		return ExeModuleKind

	default:
	}

	// Default to library
	return LibModuleKind
}

// determineVersioned determines if a module should be versioned based on its kind
func determineVersioned(kind ModuleKind) bool {
	// Test modules are not versioned
	return kind != TestModuleKind
}

// parseGoMod reads and parses a go.mod file
func parseGoMod(goModPath dt.Filepath) (mf *modfile.File, err error) {
	var content []byte

	content, err = goModPath.ReadFile()
	if err != nil {
		goto end
	}

	// Use ParseLax for syntax-only validation (go.mod may not be buildable during local dev)
	mf, err = modfile.ParseLax(string(goModPath), content, nil)
	if err != nil {
		goto end
	}

end:
	return mf, err
}

// DiscoverModules discovers Gomion-managed modules starting at rootDir.
//
// rootDir is typically a path inside a "root repo" that Gomion knows how
// to locate based on existing behavior (Phase 1 / v0). DiscoverModules
// uses .gomion/config.json plus each module's go.mod to build a ModuleSet.
//
// If the repo's .gomion/config.json contains a requires field, DiscoverModules
// will recursively discover modules from those repos as well, using BFS traversal
// with a visited set to avoid circular dependencies.
func DiscoverModules(rootDir string) (ms *ModuleSet, err error) {
	var dirPath dt.DirPath
	var repoRoot dt.DirPath
	var visited map[string]bool
	var queue []dt.DirPath
	var currentRepo dt.DirPath
	var reqPath dt.TildeDirPath
	var reqDir dt.DirPath

	// Parse the root directory
	dirPath = dt.DirPath(rootDir)

	// Find the repo root
	repoRoot, err = findRepoRootFromDir(dirPath)
	if err != nil {
		goto end
	}

	// Create module set
	ms = NewModuleSet()

	// Initialize BFS queue and visited set
	visited = make(map[string]bool)
	queue = []dt.DirPath{repoRoot}
	visited[string(repoRoot)] = true

	// BFS traversal
	for len(queue) > 0 {
		// Dequeue
		currentRepo = queue[0]
		queue = queue[1:]

		// Discover modules in current repo
		err = discoverSingleRepoModules(currentRepo, ms)
		if err != nil {
			goto end
		}

		// Load requires from current repo's config
		var repoConfig RepoConfig
		var store cfgstore.ConfigStore

		store = cfgstore.NewConfigStore(cfgstore.ProjectConfigDirType, cfgstore.ConfigStoreArgs{
			ConfigSlug:  gomion.ConfigSlug,
			RelFilepath: gomion.ConfigFile,
			DirsProvider: &cfgstore.DirsProvider{
				ProjectDirFunc: func() (dt.DirPath, error) {
					return currentRepo, nil
				},
			},
		})

		err = store.LoadJSON(&repoConfig)
		if err != nil {
			// If config doesn't exist or can't be loaded, skip requires processing
			err = nil
			continue
		}

		// Add required repos to queue if not visited
		for _, req := range repoConfig.Requires {
			reqPath = req.Path

			// Parse and expand path (handles tilde)
			reqDir, err = dt.ParseDirPath(string(reqPath))
			if err != nil {
				// Skip invalid paths
				err = nil
				continue
			}

			// Skip if already visited
			if visited[string(reqDir)] {
				continue
			}

			// Check if repo has .gomion/config.json
			var reqStore cfgstore.ConfigStore
			reqStore = cfgstore.NewConfigStore(cfgstore.ProjectConfigDirType, cfgstore.ConfigStoreArgs{
				ConfigSlug:  gomion.ConfigSlug,
				RelFilepath: gomion.ConfigFile,
				DirsProvider: &cfgstore.DirsProvider{
					ProjectDirFunc: func() (dt.DirPath, error) {
						return reqDir, nil
					},
				},
			})

			if !reqStore.Exists() {
				// Skip repos without .gomion/config.json
				continue
			}

			// Add to queue and mark as visited
			queue = append(queue, reqDir)
			visited[string(reqDir)] = true
		}
	}

end:
	return ms, err
}

// discoverSingleRepoModules discovers modules within a single repo
func discoverSingleRepoModules(repoRoot dt.DirPath, ms *ModuleSet) (err error) {
	var store cfgstore.ConfigStore
	var repoConfig RepoConfig
	var relDir dt.PathSegments
	var moduleDir dt.DirPath
	var goModPath dt.Filepath
	var exists bool
	var mf *modfile.File
	var module *Module
	var require *modfile.Require

	// Create config store for this repo
	store = cfgstore.NewConfigStore(cfgstore.ProjectConfigDirType, cfgstore.ConfigStoreArgs{
		ConfigSlug:  gomion.ConfigSlug,
		RelFilepath: gomion.ConfigFile,
		DirsProvider: &cfgstore.DirsProvider{
			ProjectDirFunc: func() (dt.DirPath, error) {
				return repoRoot, nil
			},
		},
	})

	// Load config
	err = store.LoadJSON(&repoConfig)
	if err != nil {
		err = fmt.Errorf("failed to load .gomion/config.json at %s: %w", repoRoot, err)
		goto end
	}

	// Process each module from config
	for moduleDir = range repoConfig.Modules {
		// Determine module directory
		moduleDir = dt.DirPathJoin(repoRoot, relDir.TrimPrefix("./"))

		// Read go.mod
		goModPath = dt.FilepathJoin(moduleDir, "go.mod")
		exists, err = goModPath.Exists()
		if err != nil {
			goto end
		}

		if !exists {
			err = fmt.Errorf("go.mod not found for module at: %s", moduleDir)
			goto end
		}

		mf, err = parseGoMod(goModPath)
		if err != nil {
			err = fmt.Errorf("failed to parse go.mod at %s: %w", goModPath, err)
			goto end
		}

		// Create module
		module = &Module{
			RepoRoot:   repoRoot,
			RelDir:     relDir,
			ModulePath: ModulePath(mf.Module.Mod.Path),
			Kind:       determineModuleKind(relDir),
			Requires:   make([]ModulePath, 0),
		}

		module.Versioned = determineVersioned(module.Kind)

		// Add to set (without dependencies first)
		ms.Add(module)
	}

	// Second pass: build dependencies
	for moduleDir = range repoConfig.Modules {
		goModPath = dt.FilepathJoin(moduleDir, "go.mod")

		mf, err = parseGoMod(goModPath)
		if err != nil {
			err = fmt.Errorf("failed to parse go.mod at %s: %w", goModPath, err)
			goto end
		}

		// Find the module we created in first pass
		module, exists = ms.Get(ModulePath(mf.Module.Mod.Path))
		if !exists {
			err = fmt.Errorf("module not found in set: %s", mf.Module.Mod.Path)
			goto end
		}

		// Add requires that are in our module set
		for _, require = range mf.Require {
			_, inSet := ms.Get(ModulePath(require.Mod.Path))
			if inSet {
				module.Requires = append(module.Requires, ModulePath(require.Mod.Path))
			}
		}
	}

end:
	return err
}

// findRepoRootFromDir finds the repo root starting from a directory
func findRepoRootFromDir(startDir dt.DirPath) (repoRoot dt.DirPath, err error) {
	var dir dt.DirPath
	var gitDir dt.DirPath
	var exists bool
	var parent dt.DirPath

	dir = startDir

	for {
		gitDir = dt.DirPathJoin(dir, ".git")
		exists, err = gitDir.Exists()
		if err != nil {
			goto end
		}

		if exists {
			repoRoot = dir
			goto end
		}

		parent = dir.Dir()
		if parent == dir {
			// Reached filesystem root without finding .git
			err = fmt.Errorf("no .git directory found starting from: %s", startDir)
			goto end
		}
		dir = parent
	}

end:
	return repoRoot, err
}
