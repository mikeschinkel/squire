package gomtui

import (
	"github.com/mikeschinkel/go-dt"
)

// File represents a file with its disposition and content for display
// and its parsed hunks
type File struct {
	Path        dt.RelFilepath
	Disposition FileDisposition
	Content     string // For display in right pane
	Hunks       []Hunk
	YOffset     int // Viewport scroll position
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
