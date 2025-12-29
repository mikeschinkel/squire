package squiresvc

import (
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-cliutil/climenu"
	"github.com/mikeschinkel/go-dt"
)

// NewComposeMode creates the Compose mode (F5)
// Generate and manage commit messages
func NewComposeMode(moduleDir dt.DirPath, writer cliutil.Writer, logger *slog.Logger) climenu.MenuMode {
	mode := &composeMode{
		modeBase: newModeBase(moduleDir),
	}

	baseMode := climenu.NewBaseMenuMode(climenu.BaseMenuModeArgs{
		ModeID:   3,
		ModeName: "Compose",
		Writer:   writer,
		MenuOptions: []climenu.MenuOption{
			{
				Name:        "Staged",
				Description: "Show staged changes",
				Handler:     mode.handleStaged,
			},
			{
				Name:        "Generate",
				Description: "Generate commit message",
				Handler:     mode.handleGenerate,
			},
			{
				Name:        "List",
				Description: "List commit candidates",
				Handler:     mode.handleList,
			},
			{
				Name:        "Merge",
				Description: "Merge candidates (future)",
				Handler:     mode.handleMerge,
			},
			{
				Name:        "Edit",
				Description: "Edit candidate",
				Handler:     mode.handleEdit,
			},
		},
	})
	baseMode.SetLogger(logger)

	mode.BaseMenuMode = baseMode
	return mode
}

// handleStaged displays staged changes
func (m *composeMode) handleStaged(args *climenu.OptionHandlerArgs) (err error) {
	m.Writer.Printf("\n=== Staged Changes ===\n")

	if len(m.StagedFiles) == 0 {
		m.Writer.Printf("No staged files.\n")
		m.Writer.Printf("Use F4 (Manage) to stage files first.\n")
		goto end
	}

	// TODO: Execute git diff --cached --stat
	// TODO: Display diff output
	m.Writer.Printf("Staged diff display not yet implemented.\n")
	m.Writer.Printf("Staged files: %d\n", len(m.StagedFiles))

end:
	return err
}

// handleGenerate generates a commit message from staged changes
func (m *composeMode) handleGenerate(args *climenu.OptionHandlerArgs) (err error) {
	// Check if staging area is empty
	if len(m.StagedFiles) == 0 {
		m.Writer.Printf("No staged files to analyze.\n")
		m.Writer.Printf("Use F4 (Manage) to stage files first.\n")
		goto end
	}

	// TODO: Compute staging hash
	// TODO: Call AI to generate commit message (commitmsg.GenerateWithAnalysis())
	// TODO: Create CommitCandidate with message, staging hash, analysis hash
	// TODO: Save candidate to .squire/candidates/{id}.json
	// TODO: Refresh active candidates
	m.Writer.Printf("Generate functionality not yet implemented.\n")
	m.Writer.Printf("This will call AI to generate a commit message.\n")

end:
	return err
}

// handleList lists all active commit candidates
func (m *composeMode) handleList(args *climenu.OptionHandlerArgs) (err error) {
	var currentHash string

	m.Writer.Printf("\n=== Commit Candidates ===\n")

	if len(m.ActiveCandidates) == 0 {
		m.Writer.Printf("No commit candidates available.\n")
		m.Writer.Printf("Use [2] Generate to create a candidate.\n")
		goto end
	}

	// TODO: Compute current staging hash
	currentHash = ComputeStagingHash(m.StagedFiles)

	// Display candidates
	for i, candidate := range m.ActiveCandidates {
		m.Writer.Printf("\n[%d] %s\n", i+1, candidate.ID)
		m.Writer.Printf("    Created: %s\n", candidate.Created.Format("2006-01-02 15:04:05"))
		m.Writer.Printf("    Message: %s\n", candidate.Message)

		// Mark stale if staging hash doesn't match
		if candidate.StagingHash != currentHash {
			m.Writer.Printf("    [STALE - staging area has changed]\n")
		}
	}

end:
	return err
}

// handleMerge merges multiple candidates (Phase 6)
func (m *composeMode) handleMerge(args *climenu.OptionHandlerArgs) (err error) {
	m.Writer.Printf("Merge functionality is planned for Phase 6 (Polish).\n")
	m.Writer.Printf("This will allow combining parts of multiple candidates.\n")
	return err
}

// handleEdit opens a candidate in $EDITOR
func (m *composeMode) handleEdit(args *climenu.OptionHandlerArgs) (err error) {
	// Check if there are candidates
	if len(m.ActiveCandidates) == 0 {
		m.Writer.Printf("No commit candidates to edit.\n")
		m.Writer.Printf("Use [2] Generate to create a candidate first.\n")
		goto end
	}

	// TODO: List candidates
	// TODO: Prompt user to select a candidate
	// TODO: Open candidate.Message in $EDITOR
	// TODO: Update candidate.Message and candidate.Modified timestamp
	// TODO: Save candidate
	m.Writer.Printf("Edit functionality not yet implemented.\n")

end:
	return err
}

// composeMode wraps BaseMenuMode and embeds modeBase
type composeMode struct {
	*climenu.BaseMenuMode
	*modeBase
}

func (m *composeMode) OnEnter(state climenu.ModeState) (err error) {
	// Refresh git status
	err = m.RefreshGitStatus()
	if err != nil {
		goto end
	}

	// Load active candidates
	err = m.LoadActiveCandidates()
	if err != nil {
		goto end
	}

end:
	return err
}

func (m *composeMode) OnExit(state climenu.ModeState) (err error) {
	// No cleanup needed
	return nil
}
