package gomtui

import (
	"context"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/bubbletree"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
)

type FileSource struct {
	RelPath dt.RelDirPath
	files   []bubbletree.File
}

func (fs *FileSource) HasFiles() bool {
	return len(fs.files) > 0
}

func (fs *FileSource) Files() []bubbletree.File {
	if fs.files == nil {
		panic("fileSource contains no files; you must call LoadSelectedFiles() before Files()")
	}
	return fs.files
}

func NewFileSource(dirPath dt.RelDirPath, files []bubbletree.File) *FileSource {
	return &FileSource{
		RelPath: dirPath,
		files:   files,
	}
}

// LoadChangedRepoFiles loads changed files frum a Git repo, optionally filtered to module scope
func LoadChangedRepoFiles(ctx context.Context, repo *gitutils.Repo, relPath dt.RelDirPath) (files []bubbletree.File, err error) {
	var changedFiles []dt.RelFilepath

	changedFiles, err = repo.GetChangedFiles(ctx, &gitutils.StatusArgs{
		FileFilter: gitutils.CreateDirPathFilter(relPath),
	})
	if err != nil {
		goto end
	}

	// Convert to File
	files = make([]bubbletree.File, len(changedFiles))
	for i, path := range changedFiles {
		files[i] = bubbletree.File{
			Path: path,
		}
	}

end:
	return files, err
}
