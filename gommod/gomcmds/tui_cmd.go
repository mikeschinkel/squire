package gomcmds

import (
	"github.com/mikeschinkel/go-cliutil"
)

var _ cliutil.CommandHandler = (*TUICmd)(nil)

// TUICmd handles showing tui information
type TUICmd struct {
	*cliutil.CmdBase
}

func init() {
	var err error

	err = cliutil.RegisterCommand(&TUICmd{
		CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
			Order:       99,
			Name:        "tui",
			Usage:       "",
			Description: "Show Text User Interface (TUI)",
		}),
	})
	if err != nil {
		panic(err)
	}
}

// Handle executes the tui command
func (c *TUICmd) Handle() (err error) {
	// Call TUI here
	return err
}
