package squirescliui

import (
	"context"
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-cliutil/climenu"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/squire/squirepkg/askai"
	"github.com/mikeschinkel/squire/squirepkg/gitutils"
	"github.com/mikeschinkel/squire/squirepkg/precommit"
	"github.com/mikeschinkel/squire/squirepkg/squiresvc"
)

// DirtyRepoModeArgs contains arguments for NewDirtyRepoMode
type DirtyRepoModeArgs struct {
	ModuleDir dt.DirPath
	Writer    cliutil.Writer
	Logger    *slog.Logger
}

// NewDirtyRepoMode creates a menu mode for dirty repository operations
func NewDirtyRepoMode(args DirtyRepoModeArgs) *climenu.BaseMenuMode {
	streamer := gitutils.NewStreamer(
		args.Writer.Writer(),
		args.Writer.ErrWriter(),
	)

	return climenu.NewBaseMenuMode(climenu.BaseMenuModeArgs{
		MenuOptions: []climenu.MenuOption{
			{
				Name:        "status",
				Description: "Run git status to see current changes",
				Handler: func(handlerArgs *climenu.OptionHandlerArgs) error {
					return streamer.Status(args.ModuleDir)
				},
			},
			{
				Name:        "stage",
				Description: "Stage files for this module (excluding nested modules)",
				Handler: func(handlerArgs *climenu.OptionHandlerArgs) error {
					err := streamer.StageModuleFiles(args.ModuleDir)
					if err == nil {
						// Show updated status after staging
						_ = streamer.Status(args.ModuleDir)
					}
					return err
				},
			},
			{
				Name:        "unstage",
				Description: "Unstage all currently staged files",
				Handler: func(handlerArgs *climenu.OptionHandlerArgs) error {
					err := streamer.UnstageAll(args.ModuleDir)
					if err == nil {
						// Show updated status after unstaging
						_ = streamer.Status(args.ModuleDir)
					}
					return err
				},
			},
			{
				Name:        "generate",
				Description: "Generate commit message using AI analysis",
				Handler: func(handlerArgs *climenu.OptionHandlerArgs) error {
					return generateCommitMessageInteractive(args.ModuleDir, args.Writer, args.Logger)
				},
			},
		},
	})
}

// CommitMessageModeArgs contains arguments for NewCommitMessageMode
type CommitMessageModeArgs struct {
	ModuleDir       dt.DirPath
	Message         *string
	AnalysisResults *precommit.Results
	Writer          cliutil.Writer
	Logger          *slog.Logger
}

// NewCommitMessageMode creates a menu mode for commit message operations
func NewCommitMessageMode(args CommitMessageModeArgs) *climenu.BaseMenuMode {
	streamer := gitutils.NewStreamer(args.Writer.Writer(), args.Writer.ErrWriter())

	options := []climenu.MenuOption{
		{
			Name:        "commit",
			Description: "Use this commit message and commit the staged changes",
			Handler: func(handlerArgs *climenu.OptionHandlerArgs) error {
				err := streamer.Commit(args.ModuleDir, *args.Message)
				if err == nil {
					// Successful commit - exit the menu
					handlerArgs.Mode.RequestExit()
				}
				return err
			},
		},
		{
			Name:        "edit",
			Description: "Edit the commit message in your editor",
			Handler: func(handlerArgs *climenu.OptionHandlerArgs) error {
				newMessage, err := EditMessage(*args.Message, args.Writer.Writer())
				if err == nil {
					*args.Message = newMessage
				}
				return err
			},
		},
		{
			Name:        "regenerate",
			Description: "Ask AI to generate a new commit message",
			Handler: func(handlerArgs *climenu.OptionHandlerArgs) error {
				agent := askai.NewAgent(askai.AgentArgs{
					Provider:       askai.NewClaudeCLIProvider(askai.DefaultClaudeCLIProviderArgs()),
					TimeoutSeconds: 60,
				})
				newMessage, err := squiresvc.RegenerateMessage(context.Background(), args.ModuleDir, args.AnalysisResults, agent, args.Writer.Writer())
				if err == nil {
					*args.Message = newMessage
				}
				return err
			},
		},
	}

	// Add conditional options if analysis results available
	if args.AnalysisResults != nil {
		options = append(options,
			climenu.MenuOption{
				Name:        "analysis",
				Description: "View full pre-commit analysis report",
				Handler: func(handlerArgs *climenu.OptionHandlerArgs) error {
					DisplayAnalysisReport(args.AnalysisResults, args.Writer.Writer())
					return nil
				},
			},
			climenu.MenuOption{
				Name:        "split",
				Description: "Get AI suggestions to split changes into multiple commits",
				Handler: func(handlerArgs *climenu.OptionHandlerArgs) error {
					err := handleMultiCommitFlowInteractive(args.ModuleDir, args.AnalysisResults, args.Writer)
					if err == nil {
						// Multi-commit flow completed - exit this menu
						handlerArgs.Mode.RequestExit()
					}
					return err
				},
			},
		)
	}

	return climenu.NewBaseMenuMode(climenu.BaseMenuModeArgs{MenuOptions: options})
}

// generateCommitMessageInteractive generates a commit message and shows the commit message menu
func generateCommitMessageInteractive(moduleDir dt.DirPath, writer cliutil.Writer, logger *slog.Logger) (err error) {
	var message string
	var analysisResults *precommit.Results
	var agent *askai.Agent

	ctx := context.Background()

	// Create askai agent
	agent = askai.NewAgent(askai.AgentArgs{
		Provider:       askai.NewClaudeCLIProvider(askai.DefaultClaudeCLIProviderArgs()),
		TimeoutSeconds: 60,
	})

	// Generate commit message with analysis
	message, analysisResults, err = squiresvc.GenerateWithAnalysis(ctx, squiresvc.GenerateWithAnalysisArgs{
		ModuleDir: moduleDir,
		Logger:    logger,
		Writer:    writer.Writer(),
		Agent:     agent,
	})
	if err != nil {
		goto end
	}

	// Show commit message menu
	err = climenu.ShowMenu(climenu.MenuArgs{
		Mode: NewCommitMessageMode(CommitMessageModeArgs{
			ModuleDir:       moduleDir,
			Message:         &message,
			AnalysisResults: analysisResults,
			Writer:          writer,
			Logger:          logger,
		}),
		Writer: writer,
	})

end:
	return err
}

// handleMultiCommitFlowInteractive handles the multi-commit splitting workflow
func handleMultiCommitFlowInteractive(moduleDir dt.DirPath, analysisResults *precommit.Results, writer cliutil.Writer) (err error) {
	var agent *askai.Agent
	var groups []precommit.CommitGroup

	ctx := context.Background()

	// Create AI agent
	agent = askai.NewAgent(askai.AgentArgs{
		Provider:       askai.NewClaudeCLIProvider(askai.DefaultClaudeCLIProviderArgs()),
		TimeoutSeconds: 120,
	})

	// Run multi-commit flow
	groups, err = precommit.RunMultiCommitFlow(ctx, precommit.MultiCommitFlowArgs{
		ModuleDir:       moduleDir,
		AnalysisResults: analysisResults,
		AIAgent:         agent,
	})
	if err != nil {
		goto end
	}

	// Display the suggested groups
	DisplayCommitGroups(groups, writer.Writer())

end:
	return err
}
