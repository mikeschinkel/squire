package gomcmds

import (
	"errors"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gomtui"
)

var _ cliutil.CommandHandler = (*TUICmd)(nil)

var tuiOpts = &struct {
	dir *string
}{
	dir: new(string),
}

// TUICmd handles launching the GRU TUI staging editor
type TUICmd struct {
	*cliutil.CmdBase
}

func init() {
	var err error

	err = cliutil.RegisterCommand(&TUICmd{
		CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
			Order:       99,
			Name:        "tui",
			Usage:       "tui [directory]",
			Description: "Launch TUI staging editor for interactive commit workflow",
			ArgDefs: []*cliutil.ArgDef{
				{
					Name:    "dir",
					Usage:   "Go Module directory ",
					Default: ".",
					String:  tuiOpts.dir,
					Example: "~/Projects/example/mygomod",
				},
			},
		}),
	})
	if err != nil {
		panic(err)
	}
}

// Handle executes the tui command
func (c *TUICmd) Handle() (err error) {
	var tui *gomtui.TUI

	// Create TUI instance with writer and logger
	tui = gomtui.New(c.Writer, c.Logger)

	var modDir dt.DirPath
	modDir, err = dt.ParseDirPath(*tuiOpts.dir)
	if errors.Is(err, dt.ErrEmpty) {
		err = nil
		modDir = "." // TODO: We need to fix defaults in cliutils.ArgDefs
	}
	if err != nil {
		goto end
	}
	// Run TUI with remaining args
	err = tui.Run(modDir)
end:
	return err
}
