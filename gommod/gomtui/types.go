package gomtui

import (
	"fmt"

	"github.com/mikeschinkel/go-dt"
)

// ViewMode represents which view the user is currently in
type ViewMode int

const (
	FileSelectionView ViewMode = iota
	TakesView
	FilesView
)

func (v ViewMode) String() string {
	switch v {
	case FileSelectionView:
		return "File Selection"
	case TakesView:
		return "Takes"
	case FilesView:
		return "Files"
	default:
		return "Unknown"
	}
}

// Pane represents which pane currently has focus
type Pane int

const (
	LeftPane Pane = iota
	MiddlePane
	RightPane
)

func (p Pane) String() string {
	switch p {
	case LeftPane:
		return "Left"
	case MiddlePane:
		return "Middle"
	case RightPane:
		return "Right"
	default:
		return "Unknown"
	}
}

// FileDisposition represents how a file should be handled in commit workflow
type FileDisposition byte

const (
	UnspecifiedDisposition FileDisposition = 0
	CommitDisposition      FileDisposition = 'c'
	OmitDisposition        FileDisposition = 'o' // Special ChangeSet - skip for takes
	GitIgnoreDisposition   FileDisposition = 'i' // Add to .gitignore
	GitExcludeDisposition  FileDisposition = 'x' // Add to .git/info/exclude
)

func (d FileDisposition) String() string {
	switch d {
	case CommitDisposition:
		return "Commit"
	case OmitDisposition:
		return "Omit"
	case GitIgnoreDisposition:
		return ".gitignore"
	case GitExcludeDisposition:
		return ".git/info/exclude"
	default:
		return "Unknown"
	}
}

// IsFileDisposition returns true if the string passed matches a valid file disposition
func IsFileDisposition(fd FileDisposition) bool {
	return fd.Key() == string(fd)
}

// Suffix returns the disposition marker as a suffix (appears after filename)
func (d FileDisposition) Suffix() string {
	return fmt.Sprintf("[%s]", d.Key())
}

// Key returns the disposition key
func (d FileDisposition) Key() (key string) {
	if d == 0 {
		return "?"
	}
	switch d {
	case CommitDisposition, OmitDisposition, GitIgnoreDisposition, GitExcludeDisposition:
		return string(d)
	}
	return "?"
}

type RGBColor string

const (
	GreenColor  RGBColor = "#00ff00"
	GrayColor   RGBColor = "#808080"
	RedColor    RGBColor = "#ff0000"
	YellowColor RGBColor = "#ffff00"
	WhiteColor  RGBColor = "#ffffff"
	SilverColor RGBColor = "#c0c0c0"
)

// Color returns the lipgloss color for the disposition
func (d FileDisposition) Color() string {
	switch d {
	case CommitDisposition:
		return string(GreenColor) // Green
	case OmitDisposition:
		return string(GrayColor) // Gray
	case GitIgnoreDisposition:
		return string(RedColor) // Red
	case GitExcludeDisposition:
		return string(YellowColor) // Yellow
	case UnspecifiedDisposition:
		fallthrough
	default:
		return string(WhiteColor) // White
	}
}

// FileWithDisposition represents a file with its disposition and content for display
type FileWithDisposition struct {
	Path        dt.RelFilepath
	Disposition FileDisposition
	Content     string // For display in right pane
}

// FileWithHunks represents a file and its parsed hunks
type FileWithHunks struct {
	Path  dt.RelFilepath
	Hunks []Hunk
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
