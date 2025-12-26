package goutils

import (
	"errors"

	"github.com/mikeschinkel/go-dt"
)

var ErrRepoRootNotFound = errors.New("repo root not found")

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
