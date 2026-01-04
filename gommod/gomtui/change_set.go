package gomtui

import (
	"fmt"
	"os"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
)

const (
	ChangeSetsDir      dt.PathSegment = "changesets"
	IndexFilename      dt.Filename    = "index"
	ChangeSetMetaFile  dt.Filename    = "meta.json"
	ChangeSetPatchFile dt.Filename    = "staged.patch"
)

var changeSetPaths = make(map[string]dt.DirPath)

// ChangeSet represents a logical group of changes within a take
type ChangeSet struct {
	ID         string
	Name       string
	Rationale  string
	Files      []dt.RelFilepath
	IndexFile  dt.Filepath // Path to GIT_INDEX_FILE
	TakeNumber int         // Which take this belongs to (1-based)
	Committed  bool
}

// changeSetsPath returns the path to the changesets directory
func (cs ChangeSet) changeSetsPath(projectRoot dt.DirPath) dt.DirPath {
	return dt.DirPathJoin3(projectRoot, gitutils.InfoPath, ChangeSetsDir)
}

// getChangeSetPath returns the cached path to a specific changeset directory
func (cs ChangeSet) changeSetPath(projectRoot dt.DirPath) dt.DirPath {
	key := fmt.Sprintf("%s\t%s", projectRoot, cs.ID)
	_, ok := changeSetPaths[key]
	if !ok {
		changeSetPaths[key] = dt.DirPathJoin(cs.changeSetsPath(projectRoot), cs.ID)
	}
	return changeSetPaths[key]
}

// CreateIndex creates a new Git index file for this ChangeSet
func (cs ChangeSet) CreateIndex(projectRoot dt.DirPath) (_ ChangeSet, err error) {
	var changeSetDir dt.DirPath
	var gitIndexPath dt.Filepath
	var exists bool

	// Get the changeset directory path
	changeSetDir = cs.changeSetPath(projectRoot)

	// Create the ChangeSet directory
	err = changeSetDir.MkdirAll(0755)
	if err != nil {
		err = NewErr(ErrGitChangeSet, dt.ErrFailedtoCreateDir, err)
		goto end
	}

	// Set the index file path
	cs.IndexFile = dt.FilepathJoin(changeSetDir, IndexFilename)

	// Initialize empty index by copying from .git/index
	// If .git/index doesn't exist, git will create a new one when GIT_INDEX_FILE is set
	gitIndexPath = dt.FilepathJoin3(projectRoot, gitutils.RepoPath, "index")
	exists, err = gitIndexPath.Exists()
	if err != nil {
		err = NewErr(ErrGit, dt.ErrFileSystem, err)
		goto end
	}
	if exists {
		// Copy existing index as starting point
		err = copyFile(gitIndexPath, cs.IndexFile)
		if err != nil {
			goto end
		}
	}
	// If .git/index doesn't exist, that's fine - git will create it

end:
	if err != nil {
		err = WithErr(err, "changeset_id", cs.ID)
	}
	return cs, err
}

// LoadIndex verifies this ChangeSet's index file exists and loads its path
func (cs ChangeSet) LoadIndex(projectRoot dt.DirPath) (_ ChangeSet, err error) {
	var changeSetDir dt.DirPath
	var exists bool

	changeSetDir = cs.changeSetPath(projectRoot)
	cs.IndexFile = dt.FilepathJoin(changeSetDir, IndexFilename)

	exists, err = cs.IndexFile.Exists()
	if err != nil {
		err = NewErr(ErrGit, dt.ErrFileSystem, err)
		goto end
	}
	if !exists {
		err = NewErr(ErrGitIndex, dt.ErrFileNotExists)
		goto end
	}

end:
	if err != nil {
		err = WithErr(err, "changeset_id", cs.ID)
	}
	return cs, err
}

// StageHunk stages a specific hunk to this ChangeSet's index file
// This uses GIT_INDEX_FILE environment variable to isolate the staging
func (cs ChangeSet) StageHunk(file dt.RelFilepath, hunk HunkHeader, repoRoot dt.DirPath) (_ ChangeSet, err error) {
	var originalIndex string

	// Set GIT_INDEX_FILE to the ChangeSet's index
	originalIndex = os.Getenv("GIT_INDEX_FILE")
	err = os.Setenv("GIT_INDEX_FILE", string(cs.IndexFile))
	if err != nil {
		err = NewErr(ErrGitIndex, err)
		goto end
	}
	defer func() {
		if originalIndex != "" {
			_ = os.Setenv("GIT_INDEX_FILE", originalIndex)
		} else {
			_ = os.Unsetenv("GIT_INDEX_FILE")
		}
	}()

	// TODO: Implement actual hunk staging
	// This will require:
	// 1. Creating a patch file for just this hunk
	// 2. Running `git apply --cached <patch>` to stage it
	// For now, this is a placeholder
	_ = file
	_ = hunk
	_ = repoRoot

	err = NewErr(dt.ErrNotImplemented)

end:
	if err != nil {
		err = WithErr(err,
			"changeset_id", cs.ID,
			"file", file,
		)
	}
	return cs, err
}

// MetaPath returns the path to this ChangeSet's metadata file
func (cs ChangeSet) MetaPath(projectRoot dt.DirPath) dt.Filepath {
	return dt.FilepathJoin(cs.changeSetPath(projectRoot), ChangeSetMetaFile)
}

// PatchPath returns the path to this ChangeSet's staged patch file (optional/informational)
func (cs ChangeSet) PatchPath(projectRoot dt.DirPath) dt.Filepath {
	return dt.FilepathJoin(cs.changeSetPath(projectRoot), ChangeSetPatchFile)
}

// copyFile copies a file from src to dst
// TODO: This functionality is in go-st (Domain Types) or if not, it should be.
func copyFile(src, dst dt.Filepath) (err error) {
	var data []byte
	var info os.FileInfo
	var perm os.FileMode

	// Read source file
	data, err = src.ReadFile()
	if err != nil {
		err = NewErr(dt.ErrFailedToReadFile, err)
		goto end
	}

	// Get source file permissions
	info, err = src.Stat()
	if err != nil {
		err = NewErr(dt.ErrFileSystem, err)
		goto end
	}
	perm = info.Mode()

	// Write to destination
	err = dst.WriteFile(data, perm)
	if err != nil {
		err = NewErr(dt.ErrFailedToWriteToFile, err)
		goto end
	}

end:
	if err != nil {
		err = WithErr(err, "src", src, "dst", dst)
	}
	return err
}
