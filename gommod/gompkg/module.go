package gompkg

import (
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
)

// CreateModuleFileFilter returns a filter for files within module directory
// The filter returns true if the file is within the specified module directory
// moduleRelPath should be relative to the git repo root (e.g., "gommod" or ".")
func CreateModuleFileFilter(moduleRelPath dt.RelDirPath) (ff gitutils.FileFilter) {
	// Prepare module prefix with trailing separator
	var modulePrefix dt.PathSegments

	// If module is current directory, all files match
	if moduleRelPath == "." || moduleRelPath == "" {
		ff = func(file dt.RelFilepath) bool {
			return true
		}
		goto end
	}

	// Ensure module path has trailing separator for prefix matching
	modulePrefix = moduleRelPath
	if !modulePrefix.HasSuffix("/") {
		modulePrefix += "/"
	}

	ff = func(file dt.RelFilepath) bool {
		// Check if file path starts with module directory
		return file.HasPrefix(modulePrefix)
	}
end:
	return ff

}

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
