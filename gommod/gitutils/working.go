package gitutils

import (
	"context"

	"github.com/mikeschinkel/go-dt"
)

// FileFilter is a function that returns true if file should be included
type FileFilter func(dt.RelFilepath) bool

// GetChangedFilesFiltered returns changed files matching the filter function
// This includes both staged and unstaged changes, filtered by the provided function
func (r *Repo) GetChangedFilesFiltered(ctx context.Context, filter FileFilter, args *StatusArgs) (filtered []dt.RelFilepath, err error) {
	var allFiles []dt.RelFilepath

	// Get all changed files
	allFiles, err = r.GetChangedFiles(ctx, args)
	if err != nil {
		goto end
	}

	// Apply filter if provided
	if filter != nil {
		filtered = make([]dt.RelFilepath, 0, len(allFiles))
		for _, file := range allFiles {
			if filter(file) {
				filtered = append(filtered, file)
			}
		}
	} else {
		filtered = allFiles
	}

end:
	return filtered, err
}

// GetWorkingDiff returns the full diff of all changes in the working directory
// This includes both staged and unstaged changes
func (r *Repo) GetWorkingDiff(ctx context.Context) (diff string, err error) {
	var stagedDiff string
	var unstagedDiff string

	// Get staged diff (what's in the index)
	stagedDiff, err = r.GetStagedDiff(ctx)
	if err != nil {
		goto end
	}

	// Get unstaged diff (working directory changes not in index)
	unstagedDiff, err = r.runGit(ctx, r.Root, "diff")
	if err != nil {
		goto end
	}

	// Combine both diffs
	if stagedDiff != "" && unstagedDiff != "" {
		diff = stagedDiff + "\n" + unstagedDiff
	} else if stagedDiff != "" {
		diff = stagedDiff
	} else {
		diff = unstagedDiff
	}

end:
	return diff, err
}
