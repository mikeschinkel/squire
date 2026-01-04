package gomtui

import (
	"errors"
)

var (
	// ErrGit represents git-related errors
	ErrGit = errors.New("git error")

	// ErrGitChangeSet represents ChangeSet-related errors
	ErrGitChangeSet = errors.New("git changeset error")

	// ErrGitIndex represents Git index file errors
	ErrGitIndex = errors.New("git index error")

	// ErrNoChangedFiles indicates no files have been changed
	ErrNoChangedFiles = errors.New("no changed files")

	// ErrFileStatFailed indicates file stat operation failed
	ErrFileStatFailed = errors.New("file stat failed")

	// ErrInvalidGitStatusCode indicates invalid git status code
	ErrInvalidGitStatusCode = errors.New("invalid git status code")
)
