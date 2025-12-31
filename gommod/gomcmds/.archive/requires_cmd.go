package _archive

//import (
//	"github.com/mikeschinkel/go-cliutil"
//)
//
//var _ cliutil.CommandHandler = (*RequiresCmd)(nil)
//
//// RequiresCmd is the parent command for requires-related subcommands
//type RequiresCmd struct {
//	*cliutil.CmdBase
//}
//
//// requiresCmd is the package-level instance for child commands to reference
//var requiresCmd = &RequiresCmd{
//	CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
//		Order:       20,
//		Name:        "requires",
//		Usage:       "requires <subcommand> [flags]",
//		Description: "Commands for analyzing module dependencies",
//	}),
//}
//
//func init() {
//	err := cliutil.RegisterCommand(requiresCmd)
//	if err != nil {
//		panic(err)
//	}
//}
//
//// Handle executes the requires command
//// This is a parent command that delegates to subcommands
//func (c *RequiresCmd) Handle() (err error) {
//	// Parent command doesn't do anything on its own
//	// User should run subcommands like "requires tree"
//	c.Writer.Printf("Use 'requires tree' to visualize module dependencies\n")
//	return nil
//}
