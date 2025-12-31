package gompkg

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
	"github.com/mikeschinkel/gomion/gommod/precommit"
)

// modeBase contains Gomion-specific state for all menu modes
// Each mode embeds this to avoid duplication
// Note: Writer and Logger are in BaseMenuMode, not here
type modeBase struct {
	ModuleDir dt.DirPath

	// Cached state (refreshed by OnEnter)
	StagedFiles      []dt.RelFilepath
	UnstagedFiles    []dt.RelFilepath
	UntrackedFiles   []dt.RelFilepath
	AnalysisResults  *precommit.Results
	AnalysisCacheKey string

	// Working state
	ActivePlanID     string
	ActivePlans      []*StagingPlan
	ActiveCandidates []*CommitCandidate

	// TODO: Add AI agent when available
	// AIAgent *askai.Agent
}

// newModeBase creates a new mode base with module directory
func newModeBase(moduleDir dt.DirPath) *modeBase {
	return &modeBase{
		ModuleDir: moduleDir,
	}
}

// GitStatus runs git status and displays the familiar output
func (m *modeBase) GitStatus(writer cliutil.Writer) (err error) {
	var streamer *gitutils.Streamer

	streamer = gitutils.NewStreamer(writer.Writer(), writer.ErrWriter())
	err = streamer.Status(m.ModuleDir)

	return err
}

// RefreshGitStatus refreshes the git status by parsing `git status --porcelain`
// and populating StagedFiles, UnstagedFiles, and UntrackedFiles slices
func (m *modeBase) RefreshGitStatus() (err error) {
	var cmd *exec.Cmd
	var out []byte
	var scanner *bufio.Scanner
	var line string
	var filename string
	var statusCode string

	// Clear existing slices
	m.StagedFiles = make([]dt.RelFilepath, 0)
	m.UnstagedFiles = make([]dt.RelFilepath, 0)
	m.UntrackedFiles = make([]dt.RelFilepath, 0)

	// Get git status --porcelain output
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = string(m.ModuleDir)
	out, err = cmd.Output()
	if err != nil {
		goto end
	}

	// Parse git status --porcelain output
	// Format: "XY filename" where X is staged status, Y is unstaged status
	// X = staged status: M=modified, A=added, D=deleted, R=renamed, C=copied
	// Y = unstaged status: M=modified, D=deleted, ?=untracked
	scanner = bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line = scanner.Text()
		if len(line) < 3 {
			continue
		}

		statusCode = line[0:2]
		filename = strings.TrimSpace(line[3:])

		// Handle renamed files (format: "R  old -> new")
		if strings.Contains(filename, " -> ") {
			parts := strings.Split(filename, " -> ")
			if len(parts) == 2 {
				filename = strings.TrimSpace(parts[1])
			}
		}

		// Check if staged (first character is not space or ?)
		if statusCode[0] != ' ' && statusCode[0] != '?' {
			m.StagedFiles = append(m.StagedFiles, dt.RelFilepath(filename))
		}

		// Check if unstaged (second character is not space)
		if statusCode[1] != ' ' {
			if statusCode[1] == '?' {
				// Untracked file
				m.UntrackedFiles = append(m.UntrackedFiles, dt.RelFilepath(filename))
			} else {
				// Modified/deleted but not staged
				m.UnstagedFiles = append(m.UnstagedFiles, dt.RelFilepath(filename))
			}
		}
	}

end:
	return err
}

// RefreshAnalysis refreshes the pre-commit analysis results
func (m *modeBase) RefreshAnalysis() (err error) {
	// TODO: Call precommit analysis and populate:
	// - m.AnalysisResults
	// - m.AnalysisCacheKey

	// For now, placeholder
	m.AnalysisResults = nil
	m.AnalysisCacheKey = ""

	return nil
}

// LoadActivePlans loads all active staging plans
func (m *modeBase) LoadActivePlans() (err error) {
	m.ActivePlans, err = ListStagingPlans(m.ModuleDir)
	return err
}

// LoadActiveCandidates loads all active commit candidates
func (m *modeBase) LoadActiveCandidates() (err error) {
	m.ActiveCandidates, err = ListActiveCandidates(m.ModuleDir)
	return err
}

// RefreshAll refreshes all cached state
func (m *modeBase) RefreshAll() (err error) {
	err = m.RefreshGitStatus()
	if err != nil {
		goto end
	}

	err = m.RefreshAnalysis()
	if err != nil {
		goto end
	}

	err = m.LoadActivePlans()
	if err != nil {
		goto end
	}

	err = m.LoadActiveCandidates()
	if err != nil {
		goto end
	}

end:
	return err
}
