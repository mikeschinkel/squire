package _archive

//import (
//	"github.com/mikeschinkel/go-cliutil"
//	"github.com/mikeschinkel/squire/squirepkg/retinue"
//)
//
//var _ cliutil.CommandHandler = (*UpdateCmd)(nil)
//
//// UpdateCmd updates .squire/config.json requires field for current repo
//type UpdateCmd struct {
//	*cliutil.CmdBase
//	dirArg string
//}
//
//func init() {
//	var cmd *UpdateCmd
//	var err error
//
//	cmd = &UpdateCmd{
//		dirArg: "",
//	}
//
//	cmd.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
//		Order:       20,
//		Name:        "update",
//		Usage:       "update [<dir>]",
//		Description: "Update .squire/config.json requires field for current repo",
//		ArgDefs: []*cliutil.ArgDef{
//			{
//				Name:     "dir",
//				Usage:    "Directory of repo to update (defaults to current directory)",
//				Required: false,
//				String:   &cmd.dirArg,
//				Example:  "~/Projects/squire",
//			},
//		},
//	})
//
//	err = cliutil.RegisterCommand(cmd)
//	if err != nil {
//		panic(err)
//	}
//}
//
//// Handle executes the update command
//func (c *UpdateCmd) Handle() (err error) {
//	var config *retinue.Config
//	var result *retinue.UpdateRepoRequiresResult
//
//	config = c.Config.(*retinue.Config)
//
//	// Update requires for the repo
//	result, err = retinue.UpdateRepoRequires(&retinue.UpdateRepoRequiresArgs{
//		DirArg: c.dirArg,
//		Writer: c.Writer,
//		Logger: c.Logger,
//	})
//	if err != nil {
//		goto end
//	}
//
//	config.Writer.Printf("Updated %s\n", result.RepoRoot)
//	config.Writer.Printf("  Found %d required repositories\n", result.RequireCount)
//
//end:
//	return err
//}
