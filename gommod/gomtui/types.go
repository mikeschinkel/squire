package gomtui

import (
	"os"
	"time"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
)

// File represents a file with its disposition and content for display
// and its parsed hunks
type File struct {
	Path        dt.RelFilepath
	Disposition FileDisposition
	Content     string // For display in right pane
	Hunks       []Hunk
	YOffset     int // Viewport scroll position

	// Cached file metadata (for directory table display)
	Metadata *FileMetadata // nil if not yet loaded
}

func (f File) IsEmpty() bool {
	return f.Path == ""
}

// Hunk represents a single diff hunk
type Hunk struct {
	Header        HunkHeader
	BaselineLines []string
	ChangeLines   []string
	AssignedToCS  string // ChangeSet ID if assigned
}

// HunkHeader represents the @@ header line of a hunk
type HunkHeader struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Context  string // The part after @@, e.g., "func Login()"
}

// FileMetadata contains cached metadata about a file for display in directory tables.
type FileMetadata struct {
	Size           int64               // File size in bytes
	ModTime        time.Time           // Modification time
	Permissions    os.FileMode         // Full permissions
	StagedChange   gitutils.ChangeType // M, A, D, R, C (from git status position 0)
	UnstagedChange gitutils.ChangeType // M, D (from git status position 1)
	Staging        gitutils.Staging    // Index, Worktree, Both
	EntryStatus    dt.EntryStatus      // File, Dir, Symlink, etc.
}

// Directory represents a directory containing changed files with summary statistics.
// Follows the same pattern as File + FileMetadata.
type Directory struct {
	Path    dt.RelDirPath // Directory path
	Files   []*File       // Child files
	Summary *DirSummary   // Cached summary statistics
}

// DirSummary contains summary statistics for a directory.
// Follows the same pattern as FileMetadata (cached data for display).
type DirSummary struct {
	// Disposition counts
	CommitCount     int
	OmitCount       int
	GitIgnoreCount  int
	GitExcludeCount int

	// Git status counts
	StagedCount    int
	UnstagedCount  int
	UntrackedCount int

	// Change type counts
	ModifiedCount int
	AddedCount    int
	DeletedCount  int
	RenamedCount  int

	// Totals
	TotalFiles int
	TotalSize  int64
}
