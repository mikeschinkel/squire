package gompkg

import (
	"errors"
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"golang.org/x/mod/modfile"
)

// DiscoverRequiresArgs contains arguments for discovering required repos
type DiscoverRequiresArgs struct {
	RepoRoot dt.DirPath
	Config   *Config
	Logger   *slog.Logger
	Writer   cliutil.Writer
}

// RepoRequirement represents a required repository dependency
type RepoRequirement struct {
	Path dt.TildeDirPath `json:"path"`
	// Future fields: note, enabled, etc.
}

// DiscoverRequires discovers required repositories based on module specs
func DiscoverRequires(args *DiscoverRequiresArgs) (requires []RepoRequirement, err error) {
	var scanDir dt.DirPath
	var moduleSpecs ModuleSpecs
	var matches []dt.TildeDirPath
	var allMatches []dt.TildeDirPath
	var errs []error

	// Skip if no module specs defined
	if len(args.Config.ModuleSpecs) == 0 {
		goto end
	}

	moduleSpecs = args.Config.ModuleSpecs
	// Scan each directory for repos
	for _, scanDir = range args.Config.ScanDirs {
		// Scan this dir for matching repos

		matches, err = moduleSpecs.ScanDir(scanDir, args.RepoRoot)
		if err != nil {
			args.Writer.Errorf("failed to scan dir %s: %v", scanDir, err)
			args.Logger.Warn("error scanning dir", "dir", scanDir, "error", err)
			err = nil
		}
		if matches == nil {
			continue
		}
		allMatches = append(allMatches, matches...)
	}

	allMatches = dtx.TildeDirPaths(allMatches).Unique()

	// Build requires array from matches
	requires = make([]RepoRequirement, 0, len(allMatches))
	for _, repoPath := range allMatches {
		requires = append(requires, RepoRequirement{
			Path: repoPath,
		})
	}
	err = CombineErrs(errs)
end:
	return requires, err
}

// extractModulePath parses a go.mod file and returns its module path
func extractModulePath(goModPath dt.Filepath) (modulePath string, err error) {
	var content []byte
	var mf *modfile.File

	content, err = goModPath.ReadFile()
	if err != nil {
		goto end
	}

	// Use ParseLax for syntax-only validation (repo may not be buildable yet)
	mf, err = modfile.ParseLax(string(goModPath), content, nil)
	if err != nil {
		goto end
	}

	if mf.Module == nil {
		err = errors.New("module not found in go.mod")
		goto end
	}

	if mf.Module.Mod.Path == "" {
		err = errors.New("module path empty in go.mod")
		goto end
	}

	modulePath = mf.Module.Mod.Path

end:
	return modulePath, err
}

var ErrRepoRootNotFound = errors.New("repo root not found")
var ErrFindRepoError = errors.New("error attempting to find repository")

// FindRepoRoot finds the repository root by looking for .git directory
func FindRepoRoot(startPath dt.DirPath) (repoRoot dt.DirPath, err error) {
	var currentPath dt.DirPath
	var gitPath dt.DirPath
	var exists bool

	currentPath, err = startPath.Abs()
	if err != nil {
		goto end
	}

	for {
		// Check if .git exists in current directory
		gitPath = currentPath.Join(".git")
		exists, err = gitPath.Exists()
		if err != nil {
			goto end
		}

		if exists {
			repoRoot = currentPath
			goto end
		}

		// Move to parent directory
		currentPath = currentPath.Dir()

		// Stop if we've reached the filesystem root
		if currentPath == currentPath.Dir() {
			err = NewErr(ErrRepoRootNotFound, "start_path", startPath)
			goto end
		}
	}

end:
	return repoRoot, err
}

//
//// findRepoRoot walks up the directory tree to find the nearest .git directory
//func findRepoRoot(startDir dt.DirPath) (repoRoot dt.DirPath, err error) {
//	var dir dt.DirPath
//	var gitDir dt.DirPath
//	var exists bool
//	var parent dt.DirPath
//
//	dir = startDir
//
//	for {
//		gitDir = dt.DirPathJoin(dir, ".git")
//		exists, err = gitDir.Exists()
//		if err != nil {
//			if !os.IsNotExist(err) {
//				err = WithErr(err, dt.ErrFileSystem)
//				goto end
//			}
//			err = NewErr(dt.ErrDirNotExistExist, dt.ErrDoesNotExist, "path", gitDir, err)
//			goto end
//		}
//
//		if exists {
//			repoRoot = dir
//			goto end
//		}
//
//		parent = dir.Dir()
//		if parent == dir {
//			// Reached filesystem root without finding .git
//			err = NewErr(dt.ErrFileNotExists, dt.ErrDoesNotExist, "start_dir", startDir)
//			goto end
//		}
//		dir = parent
//	}
//
//end:
//	if err != nil {
//		err = WithErr(err, ErrRepoRoot)
//	}
//	return repoRoot, err
//}
