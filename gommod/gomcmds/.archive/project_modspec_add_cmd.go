package _archive

//import (
//	"strings"
//
//	"github.com/mikeschinkel/go-cliutil"
//	"github.com/mikeschinkel/go-dt"
//	"github.com/mikeschinkel/gomion/gommod/gomcfg"
//)
//
//var _ cliutil.CommandHandler = (*ModspecAddCmd)(nil)
//
//// ModspecAddCmd adds a module spec pattern to user config
//type ModspecAddCmd struct {
//	*cliutil.CmdBase
//	pattern string
//}
//
//func init() {
//	cmd := &ModspecAddCmd{}
//
//	cmd.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
//		Order:       1,
//		Name:        "add",
//		Usage:       "add <pattern>",
//		Description: "Add a module spec pattern to user configuration",
//		ArgDefs: []*cliutil.ArgDef{
//			{
//				Name:     "pattern",
//				Usage:    "Module pattern to add (e.g., github.com/xmlui-org/*)",
//				Required: true,
//				String:   &cmd.pattern,
//				Example:  "github.com/mikeschinkel/go-*",
//			},
//		},
//	})
//
//	err := cliutil.RegisterCommand(cmd, modspecCmd)
//	if err != nil {
//		panic(err)
//	}
//}
//
//// Handle executes the modspec add command
//func (c *ModspecAddCmd) Handle() (err error) {
//	//var config *gompkg.Config
//	var rootConfig *gomcfg.RootConfigV1
//	var pattern string
//	var specs []string
//	var spec string
//	var isDuplicate bool
//
//	//config = c.Config.(*gompkg.Config)
//
//	// Validate and normalize pattern
//	pattern = strings.TrimSpace(c.pattern)
//	if pattern == "" {
//		err = NewErr(ErrCommand, ErrProject, ErrModspec, dt.ErrInvalid, "reason", "pattern cannot be empty")
//		goto end
//	}
//
//	// Load user-level config
//	rootConfig, err = gomcfg.LoadRootConfigV1(gomcfg.LoadRootConfigV1Args{
//		AppInfo: c.AppInfo,
//		Options: c.Options,
//	})
//	if err != nil {
//		err = NewErr(ErrCommand, ErrProject, ErrModspec, ErrConfigLoad, err)
//		goto end
//	}
//
//	// Check for duplicates
//	specs = rootConfig.ModuleSpecs
//	for _, spec = range specs {
//		if spec == pattern {
//			isDuplicate = true
//			goto end
//		}
//	}
//
//	// Add new pattern
//	rootConfig.ModuleSpecs = append(rootConfig.ModuleSpecs, pattern)
//
//	// Save updated config
//	err = gomcfg.SaveRootConfigV1(rootConfig, gomcfg.SaveRootConfigV1Args{
//		AppInfo: c.AppInfo,
//	})
//	if err != nil {
//		err = NewErr(ErrCommand, ErrProject, ErrModspec, ErrConfigSave, err)
//		goto end
//	}
//
//	c.Writer.Printf("Added module spec pattern: %s\n", pattern)
//
//end:
//	if isDuplicate {
//		c.Writer.Printf("Pattern already exists: %s\n", pattern)
//	}
//	return err
//}
