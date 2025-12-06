package squirecmds

import (
	"errors"
)

// Layer sentinels
var (
	ErrCommand = errors.New("command")
	ErrScan    = errors.New("scan")
	ErrInit    = errors.New("init")
)

// Category sentinels
var (
	ErrRepoRoot       = errors.New("repo root")
	ErrAlreadyManaged = errors.New("already managed")
	ErrParsing        = errors.New("parsing")
	ErrGrouping       = errors.New("grouping")
	ErrInitialization = errors.New("initialization")
	ErrFileOperation  = errors.New("file operation")
)
