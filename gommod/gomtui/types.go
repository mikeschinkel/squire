package gomtui

import (
	"os"
	"time"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/bubbletree"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
)

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

// FileMeta contains cached metadata about a file for display in directory tables.
type FileMeta struct {
	Size           int64               // File size in bytes
	ModTime        time.Time           // Modification time
	Permissions    os.FileMode         // Full permissions
	StagedChange   gitutils.ChangeType // M, A, D, R, C (from git status position 0)
	UnstagedChange gitutils.ChangeType // M, D (from git status position 1)
	Staging        gitutils.Staging    // Index, Worktree, Both
	EntryStatus    dt.EntryStatus      // File, Dir, Symlink, etc.
}

// Directory represents a directory containing changed files with summary statistics.
// Follows the same pattern as File + FileMeta.
type Directory struct {
	RelPath dt.RelDirPath      // Directory path
	Files   []*bubbletree.File // Child files
}
