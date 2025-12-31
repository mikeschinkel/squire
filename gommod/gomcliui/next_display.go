package gomcliui

import (
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gompkg"
)

// DisplayNextResult formats and displays the engine result for the next command
func DisplayNextResult(startDir dt.DirPath, result *gompkg.EngineResult, writer cliutil.Writer) {
	isDirty := result.StagedFiles > 0 || result.UnstagedFiles > 0 || result.UntrackedFiles > 0
	hasMissingTags := len(result.MissingRemoteTags) > 0

	// Header
	writer.Printf("\nAnalyzing dependency graph:\n\n")

	// Starting directory with full details
	writer.Printf("Dependent repo:\n")
	writer.Printf("- Dir:    %s\n", startDir.ToTilde(dt.OrFullPath))
	if result.StartBranch != "" {
		writer.Printf("- Branch: %s\n", result.StartBranch)
	}
	if result.StartRemote.IsValid() {
		writer.Printf("- Remote: %s\n", result.StartRemote.Name)
	}
	writer.Printf("\n")

	// Leaf module found
	writer.Printf("Leaf-most dependency found:\n")
	writer.Printf("- Module: %s/go.mod\n", result.LeafModuleDir.ToTilde(dt.OrFullPath))
	writer.Printf("- Repo:   %s\n", result.LeafRepoDir.ToTilde(dt.OrFullPath))

	// Git status
	if result.Branch != "" {
		writer.Printf("- Branch: %s\n", result.Branch)
	}
	if result.Remote.IsValid() {
		writer.Printf("- Remote: %s\n", result.Remote.Name)
	}

	// Status assessment and actions
	switch {
	case hasMissingTags:
		writer.Printf("- Status: Fetched remote tags\n\n")
		DisplayMissingTagsActions(result, writer)

	case isDirty:
		writer.Printf("- Status: Repo DIRTY\n\n")
		DisplayDirtyActions(startDir, result, writer)

	case result.Ahead > 0:
		writer.Printf("- Status: Ahead of upstream\n\n")
		DisplayAheadActions(result, writer)

	default:
		writer.Printf("- Status: Clean\n")
		writer.Printf("- Verdict: %s\n", result.Verdict)
		writer.Printf("- Reason:  %s\n\n", result.VerdictReason)
	}
}

// DisplayMissingTagsActions shows that tags were fetched from remote
func DisplayMissingTagsActions(result *gompkg.EngineResult, writer cliutil.Writer) {
	writer.Printf("Action Taken:\n")
	writer.Printf("1. Fetched remote tags:\n")
	writer.Printf("  - Fetched tags: %v\n", result.MissingRemoteTags)
	writer.Printf("  - Note: These tags were likely created by GitHub Actions\n")
	writer.Printf("\nRerun:\n")
	writer.Printf("  - gomion next (to re-analyze with updated tags)\n\n")
}

// DisplayDirtyActionsArgs contains arguments for DisplayDirtyActions
type DisplayDirtyActionsArgs struct {
	StartDir              dt.DirPath
	Result                *gompkg.EngineResult
	Writer                cliutil.Writer
	HandleInteractive     func(moduleDir dt.DirPath) error
	ShouldShowInteractive bool
}

// DisplayDirtyActions shows actions needed for dirty repo
func DisplayDirtyActions(startDir dt.DirPath, result *gompkg.EngineResult, writer cliutil.Writer) {
	DisplayDirtyActionsWithInteractive(DisplayDirtyActionsArgs{
		StartDir:              startDir,
		Result:                result,
		Writer:                writer,
		HandleInteractive:     nil,
		ShouldShowInteractive: false,
	})
}

// DisplayDirtyActionsWithInteractive shows actions needed for dirty repo with optional interactive menu
func DisplayDirtyActionsWithInteractive(args DisplayDirtyActionsArgs) {
	actionNum := 1

	// TODO Can we dynamically determine singular or plural rather than "(s)?"
	args.Writer.Printf("Action(s) needed:\n")

	// Pull if behind
	if args.Result.Behind > 0 {
		args.Writer.Printf("%d. Pull:\n", actionNum)
		args.Writer.Printf("  - %d commits\n", args.Result.Behind)
		actionNum++
	}

	// Resolve dirty files
	args.Writer.Printf("%d. Resolve:\n", actionNum)
	if args.Result.StagedFiles > 0 {
		args.Writer.Printf("  - %d staged files\n", args.Result.StagedFiles)
	}
	if args.Result.UnstagedFiles > 0 {
		args.Writer.Printf("  - %d unstaged files\n", args.Result.UnstagedFiles)
	}
	if args.Result.UntrackedFiles > 0 {
		args.Writer.Printf("  - %d untracked files\n", args.Result.UntrackedFiles)
	}

	// Interactive menu if requested
	if args.ShouldShowInteractive && args.HandleInteractive != nil && cliutil.IsInteractive() {
		err := args.HandleInteractive(args.Result.LeafModuleDir)
		if err != nil {
			args.Writer.Printf("Interactive menu error: %v\n", err)
		}
	}
}

// DisplayAheadActions shows actions needed when ahead of upstream
func DisplayAheadActions(result *gompkg.EngineResult, writer cliutil.Writer) {
	// TODO Can we dynamically determine singular or plural rather than "(s)?"
	writer.Printf("Action(s) needed:\n")
	writer.Printf("1. Push:\n")
	writer.Printf("  - %d commits\n\n", result.Ahead)
}
