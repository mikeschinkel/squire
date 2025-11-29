package squirecmds

import (
	"strings"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/squire/squirepkg/common"
)

var _ cliutil.CommandHandler = (*HelpCmd)(nil)

// HelpCmd handles showing help information
type HelpCmd struct {
	*cliutil.CmdBase
}

func init() {
	var err error

	err = cliutil.RegisterCommand(&HelpCmd{
		CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
			Order:       99,
			Name:        "help",
			Usage:       "help [command]",
			Description: "Show help information",
		}),
	})
	if err != nil {
		panic(err)
	}
}

// Handle executes the help command
func (c *HelpCmd) Handle() (err error) {
	config := c.Config.(*common.Config)
	switch {
	case len(c.Args) == 0:
		fallthrough
	case strings.ToLower(c.Args[0]) == "help":
		err = cliutil.ShowMainHelp(cliutil.UsageArgs{
			AppInfo: config.AppInfo,
			Writer:  config.Writer,
		})
	default:
		err = cliutil.ShowCmdHelp(c.Args[0], cliutil.UsageArgs{
			AppInfo: config.AppInfo,
			Writer:  config.Writer,
		})
	}
	return err
}
