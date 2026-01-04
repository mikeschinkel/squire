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

	// Cached file metadata (for directory table display)
	metadata *FileMeta // nil if not yet loaded
}
