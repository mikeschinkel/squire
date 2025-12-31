package gomcmds

import (
	"errors"
)

// Layer sentinels
var (
	ErrCommand  = errors.New("command")
	ErrScan     = errors.New("scan")
	ErrInit     = errors.New("init")
	ErrRequires = errors.New("requires")
	ErrTree     = errors.New("tree")
	ErrProject  = errors.New("project")
	ErrModspec  = errors.New("modspec")
)

// Category sentinels
var (
	ErrRepoRoot       = errors.New("repo root")
	ErrAlreadyManaged = errors.New("already managed")
	ErrParsing        = errors.New("parsing")
	ErrGrouping       = errors.New("grouping")
	ErrInitialization = errors.New("initialization")
	ErrFileOperation  = errors.New("file operation")
	ErrInvalidFlags   = errors.New("invalid flags")
	ErrMarkerNotFound = errors.New("marker not found")
	ErrFileWrite      = errors.New("file write")
	ErrDuplicate      = errors.New("duplicate")
	ErrConfigLoad     = errors.New("config load")
	ErrConfigSave     = errors.New("config save")
)
