package gitutils

import "errors"

var (
	ErrNotGitRepo            = errors.New("not a git repository")
	ErrNoSemverTags          = errors.New("no semver tags found")
	ErrNoReachableSemverTags = errors.New("no reachable semver tags found")
)
