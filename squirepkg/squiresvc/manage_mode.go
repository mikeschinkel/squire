package squiresvc

import (
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/squire/squirepkg/gitutils"
)

// NewManageMode creates the Manage mode (F4)
// Manage staging area and groups
func NewManageMode(moduleDir dt.DirPath, writer cliutil.Writer, logger *slog.Logger) cliutil.MenuMode {
	mode := &manageMode{
		modeBase: newModeBase(moduleDir),
	}

	baseMode := cliutil.NewBaseMenuMode(cliutil.BaseMenuModeArgs{
		ModeID:   2,
		ModeName: "Manage",
		Writer:   writer,
		MenuOptions: []cliutil.MenuOption{
			{
				Name:        "Stage",
				Description: "Stage a group (exclusive)",
				Handler:     mode.handleStage,
			},
			{
				Name:        "Unstage",
				Description: "Unstage all changes",
				Handler:     mode.handleUnstage,
			},
			{
				Name:        "Group",
				Description: "AI grouping suggestions",
				Handler:     mode.handleGroup,
			},
			{
				Name:        "Split",
				Description: "Split hunks (future)",
				Handler:     mode.handleSplit,
			},
		},
	})
	baseMode.SetLogger(logger)

	mode.BaseMenuMode = baseMode
	return mode
}

// handleStage stages a group exclusively
// Unstages all → stages group's files/hunks → creates snapshot
func (m *manageMode) handleStage(args *cliutil.OptionHandlerArgs) (err error) {
	var plans []*StagingPlan
	var streamer *gitutils.Streamer

	// If no plans exist, create a default plan with all changes
	if len(m.ActivePlans) == 0 {
		plans = []*StagingPlan{m.createDefaultPlan()}
	} else {
		plans = m.ActivePlans
	}

	// TODO: List available plans and prompt user to select one
	// For now, use the first plan (or default)
	if len(plans) == 0 {
		m.Writer.Printf("No changes to stage.\n")
		goto end
	}

	m.Writer.Printf("\nStaging: %s\n", plans[0].Name)
	if plans[0].Description != "" {
		m.Writer.Printf("Description: %s\n", plans[0].Description)
	}

	// TODO: Create snapshot before staging

	// Stage module files using existing gitutils functionality
	streamer = gitutils.NewStreamer(m.Writer.Writer(), m.Writer.ErrWriter())
	err = streamer.StageModuleFiles(m.ModuleDir)
	if err != nil {
		goto end
	}

	// TODO: Create snapshot after staging

	// Refresh git status to reflect changes
	err = m.RefreshGitStatus()
	if err != nil {
		goto end
	}

	// Show git status so user can see what was staged
	m.Writer.Printf("\n")
	err = m.GitStatus(m.Writer)

end:
	return err
}

// createDefaultPlan creates a default staging plan with all module changes
func (m *manageMode) createDefaultPlan() *StagingPlan {
	var allFiles []FilePatchRange

	// Add all unstaged files
	for _, file := range m.UnstagedFiles {
		allFiles = append(allFiles, FilePatchRange{
			Path:     file,
			AllLines: true,
		})
	}

	// Add all untracked files
	for _, file := range m.UntrackedFiles {
		allFiles = append(allFiles, FilePatchRange{
			Path:     file,
			AllLines: true,
		})
	}

	return &StagingPlan{
		Name:        "All Changes",
		Description: "All changes in this module",
		Files:       allFiles,
		IsDefault:   true,
	}
}

// handleUnstage unstages all changes
func (m *manageMode) handleUnstage(args *cliutil.OptionHandlerArgs) (err error) {
	var streamer *gitutils.Streamer

	// Check if staging area is empty
	if len(m.StagedFiles) == 0 {
		m.Writer.Printf("No staged files to unstage.\n")
		goto end
	}

	// TODO: Show confirmation dialog
	// TODO: Create snapshot before unstaging

	// Unstage all changes using existing gitutils functionality
	streamer = gitutils.NewStreamer(m.Writer.Writer(), m.Writer.ErrWriter())
	err = streamer.UnstageAll(m.ModuleDir)
	if err != nil {
		goto end
	}

	// Refresh git status to reflect changes
	err = m.RefreshGitStatus()
	if err != nil {
		goto end
	}

	// Show git status so user can see what was unstaged
	m.Writer.Printf("\n")
	err = m.GitStatus(m.Writer)

end:
	return err
}

// handleGroup shows AI grouping suggestions
func (m *manageMode) handleGroup(args *cliutil.OptionHandlerArgs) (err error) {
	// Check if there are unstaged files
	if len(m.UnstagedFiles) == 0 && len(m.UntrackedFiles) == 0 {
		m.Writer.Printf("No unstaged or untracked files to group.\n")
		m.Writer.Printf("\nTip: Make some changes first, then use F3 (Explore) to see them.\n")
		goto end
	}

	// TODO: Run pre-commit analysis if needed
	// TODO: Generate AI grouping takes (3 perspectives)
	// TODO: Display takes (text-based for now, MM UI in Phase 4)
	// TODO: Prompt user to select a take
	// TODO: Save selected groups as StagingPlans
	m.Writer.Printf("Group functionality not yet implemented.\n")
	m.Writer.Printf("This will generate 3 AI grouping suggestions.\n")

end:
	return err
}

// handleSplit splits hunks for line-level assignment (Phase 5)
func (m *manageMode) handleSplit(args *cliutil.OptionHandlerArgs) (err error) {
	m.Writer.Printf("Split functionality is planned for Phase 5 (MM UI).\n")
	m.Writer.Printf("This will provide JetBrains-style line-level grouping.\n")
	return err
}

// manageMode wraps BaseMenuMode and embeds modeBase
type manageMode struct {
	*cliutil.BaseMenuMode
	*modeBase
}

func (m *manageMode) OnEnter(state cliutil.ModeState) (err error) {
	// Refresh git status
	err = m.RefreshGitStatus()
	if err != nil {
		goto end
	}

	// Load active plans
	err = m.LoadActivePlans()
	if err != nil {
		goto end
	}

end:
	return err
}

func (m *manageMode) OnExit(state cliutil.ModeState) (err error) {
	// No cleanup needed
	return nil
}
