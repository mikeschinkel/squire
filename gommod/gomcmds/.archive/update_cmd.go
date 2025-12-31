package _archive

//import (
//	"github.com/mikeschinkel/go-cliutil"
//	"github.com/mikeschinkel/gomion/gommod/gompkg"
//)
//
//var _ cliutil.CommandHandler = (*UpdateCmd)(nil)
//
//// UpdateCmd updates .gomion/config.json requires field for current repo
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
//		Description: "Update .gomion/config.json requires field for current repo",
//		ArgDefs: []*cliutil.ArgDef{
//			{
//				Name:     "dir",
//				Usage:    "Directory of repo to update (defaults to current directory)",
//				Required: false,
//				String:   &cmd.dirArg,
//				Example:  "~/Projects/gomion",
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
//	var config *gompkg.Config
//	var result *gompkg.UpdateRepoRequiresResult
//
//	config = c.Config.(*gompkg.Config)
//
//	// Update requires for the repo
//	result, err = gompkg.UpdateRepoRequires(&gompkg.UpdateRepoRequiresArgs{
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
