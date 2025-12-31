package gomcmds

import (
	"context"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-cliutil/climenu"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gomcliui"
	"github.com/mikeschinkel/gomion/gommod/gompkg"
)

var _ cliutil.CommandHandler = (*NextCmd)(nil)

var nextOpts = &struct {
	dir *string
}{
	dir: new(string),
}

// emptyModeState is a dummy ModeState implementation
// Each mode has its own embedded state (modeBase), so this is unused
type emptyModeState struct{}

func (emptyModeState) ModeState() {}

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
	var result *gompkg.EngineResult
	var config *gompkg.Config
	var startDir string
	var startDirPath dt.DirPath
	var engine *gompkg.ReleaseEngine

	ctx := context.Background()
	config = c.Config.(*gompkg.Config)

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
	engine = gompkg.NewReleaseEngine(gompkg.EngineArgs{
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
	gomcliui.DisplayNextResult(startDirPath, result, c.Writer)

	// Handle interactive menu for dirty repos
	if cliutil.IsInteractive() {
		isDirty := result.StagedFiles > 0 || result.UnstagedFiles > 0 || result.UntrackedFiles > 0
		if isDirty {
			// TODO: Plan to change to auto-registration like how commands are
			//  auto-registered in gommod/gomcmds.

			// Create mode manager
			// Note: ModeState is unused since each mode has its own embedded state (modeBase)
			var state emptyModeState
			manager := climenu.NewModeManager(state, c.Writer, c.Logger)

			// Register modes (each mode creates its own state via modeBase)
			err = manager.RegisterMode(0, gompkg.NewMainMode(result.LeafModuleDir, c.Writer, c.Logger))
			if err != nil {
				goto end
			}

			err = manager.RegisterMode(1, gompkg.NewExploreMode(result.LeafModuleDir, c.Writer, c.Logger))
			if err != nil {
				goto end
			}

			err = manager.RegisterMode(2, gompkg.NewManageMode(result.LeafModuleDir, c.Writer, c.Logger))
			if err != nil {
				goto end
			}

			err = manager.RegisterMode(3, gompkg.NewComposeMode(result.LeafModuleDir, c.Writer, c.Logger))
			if err != nil {
				goto end
			}

			// Set initial mode to Main (F2)
			err = manager.SwitchMode(0)
			if err != nil {
				goto end
			}

			// Run modal menu
			err = climenu.ShowMultiModeMenu(climenu.MultiModeMenuArgs{
				Manager: manager,
				Writer:  c.Writer,
			})
			if err != nil {
				c.Writer.Printf("Interactive menu error: %v\n", err)
			}
		}
	}

end:
	return err
}
