package squirecmds

import (
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/squire/squirepkg/retinue"
)

var _ cliutil.CommandHandler = (*PlanCmd)(nil)

var planOpts = &struct {
	dir *string
}{
	dir: new(string),
}

// PlanCmd creates .squire/config.json files for repos
type PlanCmd struct {
	*cliutil.CmdBase
}

func init() {
	err := cliutil.RegisterCommand(&PlanCmd{
		CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
			//Order:       5,
			Name: "plan",
			//Usage:       "plan [<dir>] | plan --file <path>",
			Description: "Initialize .squire/config.json for repos (scans directory or reads from file)",
			ArgDefs: []*cliutil.ArgDef{
				{
					Name:     "dir",
					Usage:    "Directory to scan and initialize (defaults=scan_dirs in ~/.config/squire)",
					Required: false,
					String:   planOpts.dir,
					Example:  "~/Projects",
				},
			},
		}),
	})
	if err != nil {
		panic(err)
	}
}

// Handle executes the plan command
func (c *PlanCmd) Handle() (err error) {
	//var config *retinue.Config
	var result *retinue.PlanResult

	// Planialize repositories
	result, err = retinue.Plan(*planOpts.dir, retinue.PlanArgs{
		Config: c.Config.(*retinue.Config),
		Logger: c.Logger,
		Writer: c.Writer,
	})
	if err != nil {
		goto end
	}

	// Print summary
	c.Writer.Printf("\nSummary:\n")
	c.Writer.Printf("  Initialized: %d repos\n", result.Initialized)
	c.Writer.Printf("  Skipped:     %d repos\n", result.Skipped)

end:
	return err
}
