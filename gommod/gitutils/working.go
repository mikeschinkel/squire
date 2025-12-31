package gitutils

import (
	"context"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// GetChangedFiles returns all changed files in the working directory
// This includes both staged and unstaged changes
func (r *Repo) GetChangedFiles(ctx context.Context) (files []dt.RelFilepath, err error) {
	var out string
	var lines []string

	// Use git status --porcelain to get all changed files
	// Format: XY filename
	// X = staged status, Y = unstaged status
	out, err = r.runGit(ctx, r.Root, "status", "--porcelain")
	if err != nil {
		goto end
	}

	lines = strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Status format: "XY filename" where XY are status codes
		// We want everything after the first 3 characters (XY + space)
		if len(line) < 4 {
			continue
		}

		filename := line[3:]
		files = append(files, dt.RelFilepath(filename))
	}

end:
	return files, err
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
