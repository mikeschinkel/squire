package gomtui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gompkg"
)

// filesLoadedMsg is sent when files are loaded asynchronously
type filesLoadedMsg struct {
	*FileSource
	fileSource *FileSource
	err        error
}

// changeDispositionMsg is emitted when a file disposition changes
type changeDispositionMsg struct {
	Node        *FileDispositionNode // Optional: if set, cascade to all descendants (directory)
	Path        dt.RelFilepath       // Single path (used when Node is nil, for files)
	Disposition FileDisposition
}

func changeDispositionCmd(cdm changeDispositionMsg) tea.Cmd {
	return func() tea.Msg {
		return cdm
	}
}

// refreshTableMsg triggers table row rebuild (async pattern)
type refreshTableMsg struct{}

// commitPlanMsg is used for all commit plan save/load operations
type commitPlanMsg struct {
	msgType BubbleTeaMsgType
	seq     int                // For debounce/staleness protection
	plan    *gompkg.CommitPlan // Loaded plan (for LoadCompleteMsgType)
	err     error              // Error from save/load operation
}

type createLayoutMsg struct{}
type resizeLayoutMsg struct {
	Width  int
	Height int
}
type scheduleSaveMsg struct{}

var createLayoutCmd = func() tea.Msg {
	return createLayoutMsg{}
}
var scheduleSaveCmd = func() tea.Msg {
	return scheduleSaveMsg{}
}
var refreshTableCmd = func() tea.Msg {
	return refreshTableMsg{}
}
