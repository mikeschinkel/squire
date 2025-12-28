package squiresvc

import (
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
)

// NewExploreMode creates the Explore mode (F3)
// Read-only exploration of changes
func NewExploreMode(moduleDir dt.DirPath, writer cliutil.Writer, logger *slog.Logger) cliutil.MenuMode {
	mode := &exploreMode{
		modeBase: newModeBase(moduleDir),
	}

	baseMode := cliutil.NewBaseMenuMode(cliutil.BaseMenuModeArgs{
		ModeID:   1,
		ModeName: "Explore",
		Writer:   writer,
		MenuOptions: []cliutil.MenuOption{
			{
				Name:        "Status",
				Description: "Display git status",
				Handler:     mode.handleStatus,
			},
			{
				Name:        "Breaking",
				Description: "Show breaking changes",
				Handler:     mode.handleBreaking,
			},
			{
				Name:        "Other Changes",
				Description: "Show other changes",
				Handler:     mode.handleOther,
			},
			{
				Name:        "Tests",
				Description: "Show test-related changes",
				Handler:     mode.handleTests,
			},
		},
	})
	baseMode.SetLogger(logger)

	mode.BaseMenuMode = baseMode
	return mode
}

// handleStatus displays git status (staged/unstaged/untracked files)
func (m *exploreMode) handleStatus(args *cliutil.OptionHandlerArgs) (err error) {
	m.Writer.Printf("\n")

	// Call git status directly and show the familiar output
	err = m.GitStatus(m.Writer)

	return err
}

// handleBreaking displays breaking changes from analysis
func (m *exploreMode) handleBreaking(args *cliutil.OptionHandlerArgs) (err error) {
	m.Writer.Printf("\n=== Breaking Changes ===\n")

	if m.AnalysisResults == nil {
		m.Writer.Printf("No analysis results available.\n")
		m.Writer.Printf("Use F4 (Manage) to run analysis first.\n")
		goto end
	}

	// TODO: Display breaking changes from m.AnalysisResults
	m.Writer.Printf("Breaking changes display not yet implemented.\n")

end:
	return err
}

// handleOther displays other changes from analysis
func (m *exploreMode) handleOther(args *cliutil.OptionHandlerArgs) (err error) {
	m.Writer.Printf("\n=== Other Changes ===\n")

	if m.AnalysisResults == nil {
		m.Writer.Printf("No analysis results available.\n")
		m.Writer.Printf("Use F4 (Manage) to run analysis first.\n")
		goto end
	}

	// TODO: Display other changes from m.AnalysisResults
	m.Writer.Printf("Other changes display not yet implemented.\n")

end:
	return err
}

// handleTests displays test-related changes
func (m *exploreMode) handleTests(args *cliutil.OptionHandlerArgs) (err error) {
	m.Writer.Printf("\n=== Test-Related Changes ===\n")

	if m.AnalysisResults == nil {
		m.Writer.Printf("No analysis results available.\n")
		m.Writer.Printf("Use F4 (Manage) to run analysis first.\n")
		goto end
	}

	// TODO: Display test-related changes from m.AnalysisResults
	m.Writer.Printf("Test changes display not yet implemented.\n")

end:
	return err
}

// exploreMode wraps BaseMenuMode and embeds modeBase
type exploreMode struct {
	*cliutil.BaseMenuMode
	*modeBase
}

func (m *exploreMode) OnEnter(state cliutil.ModeState) (err error) {
	// Refresh git status
	err = m.RefreshGitStatus()
	if err != nil {
		goto end
	}

	// Refresh analysis
	err = m.RefreshAnalysis()
	if err != nil {
		goto end
	}

end:
	return err
}

func (m *exploreMode) OnExit(state cliutil.ModeState) (err error) {
	// No cleanup needed
	return nil
}
