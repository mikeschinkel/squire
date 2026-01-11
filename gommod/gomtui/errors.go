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

	// ErrInvalidFileDisposition indicates an invalid file disposition value
	ErrInvalidFileDisposition = errors.New("invalid file disposition")
)
