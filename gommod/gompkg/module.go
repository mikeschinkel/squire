package gompkg

import (
	"github.com/mikeschinkel/go-dt"
)

// AutoDetectModule finds go.mod and returns module directory
// Searches from startDir upward until a go.mod file is found
func AutoDetectModule(startDir dt.DirPath) (moduleDir dt.DirPath, err error) {
	var current dt.DirPath
	var goModPath dt.Filepath
	var found bool

	current = startDir

	// Search upward from startDir
	for {
		// Check if go.mod exists in current directory
		goModPath = dt.FilepathJoin(current, "go.mod")

		found, err = goModPath.Exists()
		if err != nil {
			err = NewErr(ErrFileOperation, goModPath.ErrKV(), err)
			goto end
		}

		if found {
			moduleDir = current
			goto end
		}

		// Move up one directory
		parent := current.Dir()

		// If we've reached the root (parent == current), stop
		if parent == current {
			err = NewErr(ErrGoModuleNotFound, startDir.ErrKV())
			goto end
		}

		current = parent
	}

end:
	return moduleDir, err
}
