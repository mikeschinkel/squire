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

// changeDispositionMsg is emitted when a file disposition changes in the table view
type changeDispositionMsg struct {
	RelFilepath dt.RelFilepath
	Disposition FileDisposition
}

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
