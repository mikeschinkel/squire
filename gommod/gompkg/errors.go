package gompkg

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
	ErrParsing        = errors.New("parsing")
	ErrGrouping       = errors.New("grouping")
	ErrInitialization = errors.New("initialization")
	ErrFileOperation  = errors.New("error during file operation")
	ErrInvalidFlags   = errors.New("invalid flags")
	ErrMarkerNotFound = errors.New("marker not found")
	ErrFileWrite      = errors.New("file write")
	ErrDuplicate      = errors.New("duplicate")
	ErrConfigLoad     = errors.New("config load")
	ErrConfigSave     = errors.New("config save")

	// ErrAlreadyManaged indicates a repository is already managed by Gomion
	ErrAlreadyManaged = errors.New("already managed")
)

var ErrCannotExtractModulePath = errors.New("cannot extract module path")

var ErrNoRepoRoot = errors.New("no repository root")

var ErrGoModuleNameNotParsed = errors.New("name for Go module not parsed")
var ErrRepoNotFoundForGoModule = errors.New("repo not found for Go module")
var ErrNotImplemented = errors.New("not implemented")
