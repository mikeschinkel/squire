package gomtui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
	"github.com/mikeschinkel/gomion/gommod/gomcfg"
)

// EditorState is the main bubbletea model for the GRU staging editor
type EditorState struct {
	CachedRepo *gitutils.CachedWorktree
	UserRepo   *gitutils.Repo
	Takes      *gomcfg.PlanTakes
	ActiveTake int // 1-based index
	ActiveCS   int // Active ChangeSet index (1-based)
	ChangeSets []*ChangeSet
	Files      []FileWithHunks
	ViewMode   ViewMode
	FocusPane  Pane
	Width      int
	Height     int
	Writer     cliutil.Writer
	Err        error
}

// ChangeSet represents a logical group of changes within a take
type ChangeSet struct {
	ID         string
	Name       string
	Rationale  string
	Files      []dt.RelFilepath
	IndexFile  dt.Filepath // Path to GIT_INDEX_FILE
	TakeNumber int         // Which take this belongs to (1-based)
	Committed  bool
}

// ViewMode represents which view the user is currently in
type ViewMode int

const (
	TakesView ViewMode = iota
	FilesView
)

func (v ViewMode) String() string {
	switch v {
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

// Ensure EditorState implements tea.Model interface
var _ tea.Model = (*EditorState)(nil)

// Init implements tea.Model
func (m EditorState) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model (will be implemented in update.go)
func (m EditorState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Placeholder - will be implemented in update.go
	return m, nil
}

// View implements tea.Model (will be implemented in view.go)
func (m EditorState) View() string {
	// Placeholder - will be implemented in view.go
	return "GRU Staging Editor - Loading..."
}
