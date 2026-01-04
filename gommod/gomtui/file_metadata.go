package gomtui

import (
	"os"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
)

// loadFileMetadata loads metadata for a single file and caches it in File.Metadata.
// Constructs full filepath from repoRoot + file.Path, then calls Stat().
// Handles errors gracefully - if file is deleted or inaccessible, returns error but doesn't panic.
func loadFileMetadata(file *File, repoRoot dt.DirPath) (err error) {
	var filepath dt.Filepath
	var info os.FileInfo

	// Construct full path
	filepath = dt.FilepathJoin(repoRoot, file.Path)

	// Get file info using dt.Filepath.Stat() method
	info, err = filepath.Stat()
	if err != nil {
		// File might be deleted after git status - return error but don't fail
		err = NewErr(
			ErrFileStatFailed,
			"filepath", filepath,
			err,
		)
		goto end
	}

	// Initialize metadata if not already present
	if file.Metadata == nil {
		file.Metadata = &FileMetadata{}
	}

	// Populate metadata from file info
	file.Metadata.Size = info.Size()
	file.Metadata.ModTime = info.ModTime()
	file.Metadata.Permissions = info.Mode()
	file.Metadata.EntryStatus = dt.GetEntryStatus(info)

end:
	return err
}

// batchLoadMetadata loads metadata for multiple files in batch.
// Calls loadFileMetadata for each file and collects errors.
// Returns error if any file failed to load metadata.
func batchLoadMetadata(files []*File, repoRoot dt.DirPath) (err error) {
	var errs []error
	var fileErr error

	for _, file := range files {
		fileErr = loadFileMetadata(file, repoRoot)
		if fileErr != nil {
			errs = append(errs, fileErr)
		}
	}

	if len(errs) > 0 {
		// Combine all errors
		err = CombineErrs(errs)
	}

	return err
}

// enrichWithGitStatus enriches file metadata with git status information.
// Updates StagedChange, UnstagedChange, and Staging fields in file.Metadata.
// If file is not in git status map, sets changes to UnknownChangeType.
func enrichWithGitStatus(file *File, gitStatus map[dt.RelFilepath]gitutils.GitFileStatus) {
	// Initialize metadata if not present
	if file.Metadata == nil {
		file.Metadata = &FileMetadata{}
	}

	// Look up git status for this file
	status, found := gitStatus[file.Path]
	if found {
		file.Metadata.StagedChange = status.StagedChange
		file.Metadata.UnstagedChange = status.UnstagedChange
		file.Metadata.Staging = status.Staging
	} else {
		// File not in git status - set to unknown
		file.Metadata.StagedChange = gitutils.UnknownChangeType
		file.Metadata.UnstagedChange = gitutils.UnknownChangeType
		file.Metadata.Staging = gitutils.NoneStaging
	}
}

// clearFileMetadataCache clears metadata cache for all files.
// Sets File.Metadata = nil for each file.
// For future "clear cache" feature.
func clearFileMetadataCache(files []*File) {
	for _, file := range files {
		file.Metadata = nil
	}
}
