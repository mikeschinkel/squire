package gitutils

import "errors"

var (
	ErrNotGitRepo            = errors.New("not a git repository")
	ErrNoSemverTags          = errors.New("no semver tags found")
	ErrNoReachableSemverTags = errors.New("no reachable semver tags found")
	ErrInvalidGitStatusCode  = errors.New("invalid git status code")
)

var (
	ErrGitInfoPathNotExist         = errors.New(".git/info path does not exist")
	ErrFailedInspectingGitInfoPath = errors.New("failed inspecting .git/info path")
	ErrFailedToGetFileSystem       = errors.New("failed to get file system")
	ErrFailedToGetGitInfoFilepath  = errors.New("failed to get .git/info filepath")
	ErrFailedToLoadGitInfoFile     = errors.New("failed to load .git/info file")
	ErrFailedToSaveGitInfoFile     = errors.New("failed to save .git/info file")
	ErrFailedToLoadJSONFile        = errors.New("failed to load JSON file")
	ErrFailedToSaveJSONFile        = errors.New("failed to save JSON file")
)
