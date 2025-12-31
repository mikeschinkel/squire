package gompkg

import (
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-cliutil/climenu"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
)

// NewMainMode creates the Main mode (F2)
// This is the starting mode where users commit code
func NewMainMode(moduleDir dt.DirPath, writer cliutil.Writer, logger *slog.Logger) climenu.MenuMode {
	mode := &mainMode{
		modeBase: newModeBase(moduleDir),
	}

	baseMode := climenu.NewBaseMenuMode(climenu.BaseMenuModeArgs{
		ModeID:   0,
		ModeName: "Main",
		Writer:   writer,
		MenuOptions: []climenu.MenuOption{
			{
				Name:        "Status",
				Description: "Display git status",
				Handler:     mode.handleStatus,
			},
			{
				Name:        "Commit",
				Description: "Commit staged changes",
				Handler:     mode.handleCommit,
			},
		},
	})
	baseMode.SetLogger(logger)

	mode.BaseMenuMode = baseMode
	return mode
}

// handleStatus displays git status (staged/unstaged/untracked files)
func (m *mainMode) handleStatus(args *climenu.OptionHandlerArgs) (err error) {
	m.Writer.Printf("\n")

	// Call git status directly and show the familiar output
	err = m.GitStatus(m.Writer)

	return err
}

// handleCommit commits the staged changes
func (m *mainMode) handleCommit(args *climenu.OptionHandlerArgs) (err error) {
	var streamer *gitutils.Streamer
	var message string

	// Check if staging area is empty
	if len(m.StagedFiles) == 0 {
		m.Writer.Printf("No staged files to commit.\n")
		m.Writer.Printf("\nTip: Press F4 (Manage) to stage files first.\n")
		goto end
	}

	// Check if there's a commit candidate
	if len(m.ActiveCandidates) == 0 {
		m.Writer.Printf("No commit message available.\n")
		m.Writer.Printf("\nTip: Press F5 (Compose) to generate a commit message first.\n")
		goto end
	}

	// Use the first active candidate's message
	message = m.ActiveCandidates[0].Message

	m.Writer.Printf("\nCommitting with message:\n%s\n\n", message)

	// TODO: Show confirmation dialog

	// Commit using existing gitutils functionality
	streamer = gitutils.NewStreamer(m.Writer.Writer(), m.Writer.ErrWriter())
	err = streamer.Commit(m.ModuleDir, message)
	if err != nil {
		goto end
	}

	m.Writer.Printf("\nCommit successful!\n")

	// TODO: Archive the used candidate

	// Refresh git status
	err = m.RefreshGitStatus()

end:
	return err
}

// mainMode wraps BaseMenuMode and embeds modeBase
type mainMode struct {
	*climenu.BaseMenuMode
	*modeBase
}

func (m *mainMode) OnEnter(state climenu.ModeState) (err error) {
	// Refresh git status
	err = m.RefreshGitStatus()
	if err != nil {
		goto end
	}

	// Refresh candidates
	err = m.LoadActiveCandidates()
	if err != nil {
		goto end
	}

end:
	return err
}

func (m *mainMode) OnExit(state climenu.ModeState) (err error) {
	// No cleanup needed
	return nil
}
