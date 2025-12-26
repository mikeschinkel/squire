package squirecmds

import (
	"context"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/squire/squirepkg/squirescliui"
	"github.com/mikeschinkel/squire/squirepkg/squiresvc"
)

var _ cliutil.CommandHandler = (*NextCmd)(nil)

var nextOpts = &struct {
	dir *string
}{
	dir: new(string),
}

// NextCmd determines the next module to next with minimal output
type NextCmd struct {
	*cliutil.CmdBase
}

func init() {
	err := cliutil.RegisterCommand(&NextCmd{
		CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
			Name:        "next",
			Description: "Determine next Go module to tackle",
			ArgDefs: []*cliutil.ArgDef{
				{
					Name:     "dir",
					Usage:    "Directory to start from (defaults to current directory)",
					Required: false,
					String:   nextOpts.dir,
					Example:  "~/Projects/myrepo",
				},
			},
		}),
	})
	if err != nil {
		panic(err)
	}
}

// Handle executes the next command
func (c *NextCmd) Handle() (err error) {
	var result *squiresvc.EngineResult
	var config *squiresvc.Config
	var startDir string
	var startDirPath dt.DirPath
	var engine *squiresvc.ReleaseEngine

	ctx := context.Background()
	config = c.Config.(*squiresvc.Config)

	// Determine starting directory for display
	startDir = *nextOpts.dir
	if startDir == "" {
		startDir = "."
	}

	// Resolve to absolute path for display
	startDirPath, err = dt.ParseDirPath(startDir)
	if err != nil {
		goto end
	}
	startDirPath, err = startDirPath.Abs()
	if err != nil {
		goto end
	}

	// Create and run the release engine (silent mode - no streaming)
	engine = squiresvc.NewReleaseEngine(squiresvc.EngineArgs{
		StartDir: *nextOpts.dir,
		RepoDirs: []string{}, // Use config scan_dirs
		Config:   config,
		Logger:   c.Logger,
		Writer:   c.Writer,
	})

	result, err = engine.Run(ctx)
	if err != nil {
		goto end
	}

	// Display human-friendly output
	squirescliui.DisplayNextResult(startDirPath, result, c.Writer.Writer())

	// Handle interactive menu for dirty repos
	if cliutil.IsInteractive() {
		isDirty := result.StagedFiles > 0 || result.UnstagedFiles > 0 || result.UntrackedFiles > 0
		if isDirty {
			err = cliutil.ShowMenu(cliutil.MenuArgs{
				Mode: squirescliui.NewDirtyRepoMode(squirescliui.DirtyRepoModeArgs{
					ModuleDir: result.LeafModuleDir,
					Writer:    c.Writer,
					Logger:    c.Logger,
				}),
				Writer: c.Writer.Writer(),
			})
			if err != nil {
				c.Writer.Printf("Interactive menu error: %v\n", err)
			}
		}
	}

end:
	return err
}
