package squirecmds

import (
	"context"
	"strings"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/squire/squirepkg/retinue"
)

var _ cliutil.CommandHandler = (*ProcessCmd)(nil)

var processOpts = &struct {
	dir *string
}{
	dir: new(string),
}

// ProcessCmd determines the next module to process with minimal output
type ProcessCmd struct {
	*cliutil.CmdBase
}

func init() {
	err := cliutil.RegisterCommand(&ProcessCmd{
		CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
			Name:        "process",
			Description: "Determine next module to process (minimal output for automation)",
			ArgDefs: []*cliutil.ArgDef{
				{
					Name:     "dir",
					Usage:    "Directory to start from (defaults to current directory)",
					Required: false,
					String:   processOpts.dir,
					Example:  "~/Projects/myrepo",
				},
			},
		}),
	})
	if err != nil {
		panic(err)
	}
}

// Handle executes the process command
func (c *ProcessCmd) Handle() (err error) {
	var result *retinue.EngineResult
	var config *retinue.Config
	var startDir string
	var startDirPath dt.DirPath

	ctx := context.Background()
	config = c.Config.(*retinue.Config)

	// Determine starting directory for display
	startDir = *processOpts.dir
	if startDir == "" {
		startDir = "."
	}

	// Resolve to absolute path for display
	startDirPath, err = dt.ParseDirPath(startDir)
	if err == nil {
		startDirPath, err = startDirPath.Abs()
		if err == nil {
			startDir = string(startDirPath)
		}
	}

	// Create and run the release engine (silent mode - no streaming)
	engine := retinue.NewReleaseEngine(retinue.EngineArgs{
		StartDir: *processOpts.dir,
		RepoDirs: []string{}, // Use config scan_dirs
		Config:   config,
		Logger:   c.Logger,
		Writer:   nil, // No streaming output
	})

	result, err = engine.Run(ctx)
	if err != nil {
		goto end
	}

	// Display human-friendly output
	c.displayResult(startDir, result)

end:
	return err
}

// displayResult formats and displays the engine result
func (c *ProcessCmd) displayResult(startDir string, result *retinue.EngineResult) {
	isDirty := result.StagedFiles > 0 || result.UnstagedFiles > 0 || result.UntrackedFiles > 0
	hasMissingTags := len(result.MissingRemoteTags) > 0

	// Header
	c.Writer.Printf("\nAnalyzing dependency graph:\n\n")

	// Starting directory with full details
	c.Writer.Printf("Dependent repo:\n")
	c.Writer.Printf("- Dir:    %s\n", tildeNotation(startDir))
	if result.StartBranch != "" {
		c.Writer.Printf("- Branch: %s\n", result.StartBranch)
	}
	if result.StartRemote.IsValid() {
		c.Writer.Printf("- Remote: %s\n", result.StartRemote.Name)
	}
	c.Writer.Printf("\n")

	// Leaf module found
	c.Writer.Printf("Leaf-most dependency found:\n")
	c.Writer.Printf("- Module: %s/go.mod\n", tildeNotation(string(result.LeafModuleDir)))
	c.Writer.Printf("- Repo:   %s\n", tildeNotation(string(result.LeafRepoDir)))

	// Git status
	if result.Branch != "" {
		c.Writer.Printf("- Branch: %s\n", result.Branch)
	}
	if result.Remote.IsValid() {
		c.Writer.Printf("- Remote: %s\n", result.Remote.Name)
	}

	// Status assessment and actions
	switch {
	case hasMissingTags:
		c.Writer.Printf("- Status: Fetched remote tags\n\n")
		c.displayMissingTagsActions(result)

	case isDirty:
		c.Writer.Printf("- Status: Repo DIRTY\n\n")
		c.displayDirtyActions(startDir, result)

	case result.Ahead > 0:
		c.Writer.Printf("- Status: Ahead of upstream\n\n")
		c.displayAheadActions(result)

	default:
		c.Writer.Printf("- Status: Clean\n")
		c.Writer.Printf("- Verdict: %s\n", result.Verdict)
		c.Writer.Printf("- Reason:  %s\n\n", result.VerdictReason)
	}
}

// displayMissingTagsActions shows that tags were fetched from remote
func (c *ProcessCmd) displayMissingTagsActions(result *retinue.EngineResult) {
	c.Writer.Printf("Action Taken:\n")
	c.Writer.Printf("1. Fetched remote tags:\n")
	c.Writer.Printf("  - Fetched tags: %v\n", result.MissingRemoteTags)
	c.Writer.Printf("  - Note: These tags were likely created by GitHub Actions\n")
	c.Writer.Printf("\nRerun:\n")
	c.Writer.Printf("  - squire process (to re-analyze with updated tags)\n\n")
}

// displayDirtyActions shows actions needed for dirty repo
func (c *ProcessCmd) displayDirtyActions(startDir string, result *retinue.EngineResult) {
	actionNum := 1

	// TODO Can we dynamically determine singular or plural rather than "(s)?"
	c.Writer.Printf("Action(s) needed:\n")

	// Pull if behind
	if result.Behind > 0 {
		c.Writer.Printf("%d. Pull:\n", actionNum)
		c.Writer.Printf("  - %d commits\n", result.Behind)
		actionNum++
	}

	// Resolve dirty files
	c.Writer.Printf("%d. Resolve:\n", actionNum)
	if result.StagedFiles > 0 {
		c.Writer.Printf("  - %d staged files\n", result.StagedFiles)
	}
	if result.UnstagedFiles > 0 {
		c.Writer.Printf("  - %d unstaged files\n", result.UnstagedFiles)
	}
	if result.UntrackedFiles > 0 {
		c.Writer.Printf("  - %d untracked files\n", result.UntrackedFiles)
	}
	actionNum++

	// Rerun instruction
	c.Writer.Printf("%d. Rerun:\n", actionNum)
	c.Writer.Printf("  - squire process %s\n\n", tildeNotation(startDir))
}

// displayAheadActions shows actions needed when ahead of upstream
func (c *ProcessCmd) displayAheadActions(result *retinue.EngineResult) {
	// TODO Can we dynamically determine singular or plural rather than "(s)?"
	c.Writer.Printf("Action(s) needed:\n")
	c.Writer.Printf("1. Push:\n")
	c.Writer.Printf("  - %d commits\n\n", result.Ahead)
}

// tildeNotation converts paths starting with home directory to use ~ notation
func tildeNotation(path string) string {
	homeDir, err := dt.UserHomeDir()
	if err != nil {
		return path
	}

	homePath := string(homeDir)
	if strings.HasPrefix(path, homePath) {
		return strings.Replace(path, homePath, "~", 1)
	}
	return path
}
