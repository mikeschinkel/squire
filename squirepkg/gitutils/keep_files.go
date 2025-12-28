package gitutils

import (
	"github.com/mikeschinkel/go-dt"
)

func KeepFiles(subdirs []dt.PathSegments) (files []dt.RelFilepath) {
	files = make([]dt.RelFilepath, len(subdirs))
	for i, dir := range subdirs {
		files[i] = dt.RelFilepathJoin(dir, KeepFile)
	}
	return files
}
