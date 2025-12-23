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

// PlanCmd displays the dependency graph for inspection
type PlanCmd struct {
	*cliutil.CmdBase
}

func init() {
	err := cliutil.RegisterCommand(&PlanCmd{
		CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
			Name:        "plan",
			Description: "Display module dependency graph",
			ArgDefs: []*cliutil.ArgDef{
				{
					Name:     "dir",
					Usage:    "Directory to scan and initialize (defaults to current directory)",
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
	var result *retinue.PlanResult

	// Run the plan command (original graph display)
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
