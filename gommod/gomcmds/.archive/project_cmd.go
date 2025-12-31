package _archive

//import (
//	"github.com/mikeschinkel/go-cliutil"
//)
//
//var _ cliutil.CommandHandler = (*ProjectCmd)(nil)
//
//// ProjectCmd is the parent command for project-related operations
//type ProjectCmd struct {
//	*cliutil.CmdBase
//}
//
//// projectCmd is the package-level instance for child commands to reference
//var projectCmd = &ProjectCmd{
//	CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
//		Order:       30,
//		Name:        "project",
//		Usage:       "project <subcommand> [flags]",
//		Description: "Commands for managing project configuration",
//	}),
//}
//
//func init() {
//	err := cliutil.RegisterCommand(projectCmd)
//	if err != nil {
//		panic(err)
//	}
//}
//
//// Handle executes the project command
//// This is a parent command that delegates to subcommands
//func (c *ProjectCmd) Handle() (err error) {
//	c.Writer.Printf("Use 'project modspec add <pattern>' to add module spec patterns\n")
//	return nil
//}
