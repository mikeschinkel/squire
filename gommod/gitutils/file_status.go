package gitutils

import (
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// GitFileStatus represents the status of a file from git status --porcelain.
// The two-character status format is: XY filename
// where X is the index status (position 0) and Y is the working tree status (position 1).
type GitFileStatus struct {
	StagedChange   ChangeType // Status in index (position 0)
	UnstagedChange ChangeType // Status in working tree (position 1)
	Staging        Staging    // Derived staging state
}

type StatusMap map[dt.RelFilepath]GitFileStatus

// ParseStatus parses git status --porcelain output into a map of file statuses.
// Returns a map from relative filepath to GitFileStatus.
// Format of git status --porcelain:
//
//	XY filename
//	where X = index status (position 0), Y = worktree status (position 1)
//
// Examples:
//
//	"MM file.txt"     -> staged M, unstaged M
//	" M file.txt"     -> no staged change, unstaged M
//	"A  newfile.txt"  -> staged A, no unstaged change
//	"?? untracked.txt"-> untracked (both positions are '?')
func ParseStatus(output string) (statuses StatusMap, err error) {
	var line string
	var lines []string
	var statusCode string
	var filepath dt.RelFilepath
	var stagedByte, unstagedByte byte
	var stagedChange, unstagedChange ChangeType
	var staging Staging

	statuses = make(StatusMap)

	// Split output into lines
	lines = strings.Split(strings.TrimSpace(output), "\n")

	for _, line = range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Git status --porcelain format: "XY filename"
		// where X and Y are status codes
		if len(line) < 3 {
			// Invalid line format - skip
			continue
		}

		statusCode = line[0:2]
		filepath = dt.RelFilepath(strings.TrimSpace(line[3:]))

		// Parse position 0 (index/staged)
		stagedByte = statusCode[0]
		stagedChange, err = ParseChangeType(stagedByte)
		if err != nil {
			goto end
		}

		// Parse position 1 (worktree/unstaged)
		unstagedByte = statusCode[1]
		unstagedChange, err = ParseChangeType(unstagedByte)
		if err != nil {
			goto end
		}

		// Determine staging state
		staging, err = ParseStaging(stagedChange, unstagedChange)
		if err != nil {
			goto end
		}

		statuses[filepath] = GitFileStatus{
			StagedChange:   stagedChange,
			UnstagedChange: unstagedChange,
			Staging:        staging,
		}
	}

end:
	return statuses, err
}
