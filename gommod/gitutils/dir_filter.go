package gitutils

import (
	"github.com/mikeschinkel/go-dt"
)

// CreateDirPathFilter returns a filter for files within a repo but constrainted to
// just one directory in that repo. The filter returns true if the file is within
// the specified directory moduleRelPath should be relative to the git
// repo root (e.g., "gommod" or ".")
func CreateDirPathFilter(dirPath dt.RelDirPath) (ff FileFilter) {
	switch {
	case dirPath == "":
		dirPath = "."
		fallthrough

	case dirPath == ".":
		// If path is current directory, all files match
		ff = func(file dt.RelFilepath) bool {
			return true
		}

	default:
		// Ensure module path has trailing separator for prefix matching
		var prefix = dirPath
		if !prefix.HasSuffix("/") {
			prefix += "/"
		}

		ff = func(file dt.RelFilepath) bool {
			// Check if file path starts with module directory
			return file.HasPrefix(prefix)
		}
	}

	return ff
}
