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
var ErrNoRepoFoundForGoModule = errors.New("no repo found for Go module")
var ErrNotImplemented = errors.New("not implemented")

//goland:noinspection GoErrorStringFormat
var ErrGoModuleNotFound = errors.New("Go module not found")

// Commit plan errors
var (
	// ErrInvalidCommitPlan indicates the commit plan data is invalid
	ErrInvalidCommitPlan = errors.New("invalid commit plan")

	// ErrFailedToSaveCommitPlan indicates failure to save commit plan
	ErrFailedToSaveCommitPlan = errors.New("failed to save commit plan")

	// ErrFailedToLoadCommitPlan indicates failure to load commit plan
	ErrFailedToLoadCommitPlan = errors.New("failed to load commit plan")

	// ErrInvalidFileDisposition indicates an invalid file disposition value
	ErrInvalidFileDisposition = errors.New("invalid file disposition")
)
