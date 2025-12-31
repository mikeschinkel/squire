package precommit

import (
	"context"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/askai"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
)

// MultiCommitFlowArgs contains arguments for running the multi-commit flow
type MultiCommitFlowArgs struct {
	ModuleDir       dt.DirPath
	AnalysisResults *Results
	AIAgent         *askai.Agent
}

// RunMultiCommitFlow analyzes staged changes and suggests commit groupings
func RunMultiCommitFlow(ctx context.Context, args MultiCommitFlowArgs) (groups []CommitGroup, err error) {
	var repo *gitutils.Repo
	var stagedFiles []dt.RelFilepath

	// Open repo and get staged files
	repo, err = gitutils.Open(args.ModuleDir)
	if err != nil {
		goto end
	}

	stagedFiles, err = repo.GetStagedFiles(ctx)
	if err != nil {
		goto end
	}

	if len(stagedFiles) == 0 {
		// No files to group - return empty
		goto end
	}

	// Get AI suggestions for grouping
	groups, err = SuggestGroupings(ctx, GroupingArgs{
		StagedFiles: stagedFiles,
		Analysis:    *args.AnalysisResults,
		AIAgent:     args.AIAgent,
	})

end:
	return groups, err
}
