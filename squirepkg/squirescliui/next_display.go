package squirescliui

import (
	"io"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/squire/squirepkg/squiresvc"
)

// DisplayNextResult formats and displays the engine result for the next command
func DisplayNextResult(startDir dt.DirPath, result *squiresvc.EngineResult, writer io.Writer) {
	isDirty := result.StagedFiles > 0 || result.UnstagedFiles > 0 || result.UntrackedFiles > 0
	hasMissingTags := len(result.MissingRemoteTags) > 0

	// Header
	dtx.Fprintf(writer, "\nAnalyzing dependency graph:\n\n")

	// Starting directory with full details
	dtx.Fprintf(writer, "Dependent repo:\n")
	dtx.Fprintf(writer, "- Dir:    %s\n", startDir.ToTilde(dt.OrFullPath))
	if result.StartBranch != "" {
		dtx.Fprintf(writer, "- Branch: %s\n", result.StartBranch)
	}
	if result.StartRemote.IsValid() {
		dtx.Fprintf(writer, "- Remote: %s\n", result.StartRemote.Name)
	}
	dtx.Fprintf(writer, "\n")

	// Leaf module found
	dtx.Fprintf(writer, "Leaf-most dependency found:\n")
	dtx.Fprintf(writer, "- Module: %s/go.mod\n", result.LeafModuleDir.ToTilde(dt.OrFullPath))
	dtx.Fprintf(writer, "- Repo:   %s\n", result.LeafRepoDir.ToTilde(dt.OrFullPath))

	// Git status
	if result.Branch != "" {
		dtx.Fprintf(writer, "- Branch: %s\n", result.Branch)
	}
	if result.Remote.IsValid() {
		dtx.Fprintf(writer, "- Remote: %s\n", result.Remote.Name)
	}

	// Status assessment and actions
	switch {
	case hasMissingTags:
		dtx.Fprintf(writer, "- Status: Fetched remote tags\n\n")
		DisplayMissingTagsActions(result, writer)

	case isDirty:
		dtx.Fprintf(writer, "- Status: Repo DIRTY\n\n")
		DisplayDirtyActions(startDir, result, writer)

	case result.Ahead > 0:
		dtx.Fprintf(writer, "- Status: Ahead of upstream\n\n")
		DisplayAheadActions(result, writer)

	default:
		dtx.Fprintf(writer, "- Status: Clean\n")
		dtx.Fprintf(writer, "- Verdict: %s\n", result.Verdict)
		dtx.Fprintf(writer, "- Reason:  %s\n\n", result.VerdictReason)
	}
}

// DisplayMissingTagsActions shows that tags were fetched from remote
func DisplayMissingTagsActions(result *squiresvc.EngineResult, writer io.Writer) {
	dtx.Fprintf(writer, "Action Taken:\n")
	dtx.Fprintf(writer, "1. Fetched remote tags:\n")
	dtx.Fprintf(writer, "  - Fetched tags: %v\n", result.MissingRemoteTags)
	dtx.Fprintf(writer, "  - Note: These tags were likely created by GitHub Actions\n")
	dtx.Fprintf(writer, "\nRerun:\n")
	dtx.Fprintf(writer, "  - squire next (to re-analyze with updated tags)\n\n")
}

// DisplayDirtyActionsArgs contains arguments for DisplayDirtyActions
type DisplayDirtyActionsArgs struct {
	StartDir            dt.DirPath
	Result              *squiresvc.EngineResult
	Writer              io.Writer
	HandleInteractive   func(moduleDir dt.DirPath) error
	ShouldShowInteractive bool
}

// DisplayDirtyActions shows actions needed for dirty repo
func DisplayDirtyActions(startDir dt.DirPath, result *squiresvc.EngineResult, writer io.Writer) {
	DisplayDirtyActionsWithInteractive(DisplayDirtyActionsArgs{
		StartDir:            startDir,
		Result:              result,
		Writer:              writer,
		HandleInteractive:   nil,
		ShouldShowInteractive: false,
	})
}

// DisplayDirtyActionsWithInteractive shows actions needed for dirty repo with optional interactive menu
func DisplayDirtyActionsWithInteractive(args DisplayDirtyActionsArgs) {
	actionNum := 1

	// TODO Can we dynamically determine singular or plural rather than "(s)?"
	dtx.Fprintf(args.Writer, "Action(s) needed:\n")

	// Pull if behind
	if args.Result.Behind > 0 {
		dtx.Fprintf(args.Writer, "%d. Pull:\n", actionNum)
		dtx.Fprintf(args.Writer, "  - %d commits\n", args.Result.Behind)
		actionNum++
	}

	// Resolve dirty files
	dtx.Fprintf(args.Writer, "%d. Resolve:\n", actionNum)
	if args.Result.StagedFiles > 0 {
		dtx.Fprintf(args.Writer, "  - %d staged files\n", args.Result.StagedFiles)
	}
	if args.Result.UnstagedFiles > 0 {
		dtx.Fprintf(args.Writer, "  - %d unstaged files\n", args.Result.UnstagedFiles)
	}
	if args.Result.UntrackedFiles > 0 {
		dtx.Fprintf(args.Writer, "  - %d untracked files\n", args.Result.UntrackedFiles)
	}

	// Interactive menu if requested
	if args.ShouldShowInteractive && args.HandleInteractive != nil && cliutil.IsInteractive() {
		err := args.HandleInteractive(args.Result.LeafModuleDir)
		if err != nil {
			dtx.Fprintf(args.Writer, "Interactive menu error: %v\n", err)
		}
	}
}

// DisplayAheadActions shows actions needed when ahead of upstream
func DisplayAheadActions(result *squiresvc.EngineResult, writer io.Writer) {
	// TODO Can we dynamically determine singular or plural rather than "(s)?"
	dtx.Fprintf(writer, "Action(s) needed:\n")
	dtx.Fprintf(writer, "1. Push:\n")
	dtx.Fprintf(writer, "  - %d commits\n\n", result.Ahead)
}
