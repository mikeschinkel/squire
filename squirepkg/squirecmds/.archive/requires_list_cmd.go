package _archive

//import (
//	"bytes"
//
//	"github.com/mikeschinkel/go-cliutil"
//	"github.com/mikeschinkel/squire/squirepkg/squiresvc"
//)
//
//var requiresListOpts = &struct {
//	DirArg *string
//	Format *string
//}{
//	DirArg: new(string),
//	Format: new(string),
//}
//
//var RequiresListFlagSet = &cliutil.FlagSet{
//	Name: "requires-list",
//	FlagDefs: []cliutil.FlagDef{
//		{
//			Name:     "format",
//			Usage:    "Output format (table, json, csv)",
//			Required: false,
//			Default:  string(squiresvc.TableOutputFormat),
//			String:   requiresListOpts.Format,
//		},
//	},
//}
//
//var _ cliutil.CommandHandler = (*RequiresListCmd)(nil)
//
//// RequiresListCmd lists modules in dependency-safe order
//type RequiresListCmd struct {
//	*cliutil.CmdBase
//}
//
//func init() {
//	// Initialize format default
//	*requiresListOpts.Format = string(squiresvc.TableOutputFormat)
//
//	err := cliutil.RegisterCommand(&RequiresListCmd{
//		CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
//			Order:       20,
//			Name:        "requires-list",
//			Usage:       "requires-list [<dir>]",
//			Description: "List modules in dependency-safe order",
//			FlagSets:    []*cliutil.FlagSet{RequiresListFlagSet},
//			ArgDefs: []*cliutil.ArgDef{
//				{
//					Name:     "dir",
//					Usage:    "Directory to analyze (defaults to current directory)",
//					Required: false,
//					Default:  "",
//					String:   requiresListOpts.DirArg,
//					Example:  ".",
//				},
//			},
//		}),
//	})
//	if err != nil {
//		panic(err)
//	}
//}
//
//// Handle executes the requires list command
//func (c *RequiresListCmd) Handle() (err error) {
//	var config *squiresvc.Config
//	var dir string
//	var ms *squiresvc.ModuleSet
//	var ordered squiresvc.Modules
//	var format squiresvc.OutputFormat
//
//	config = c.Config.(*squiresvc.Config)
//
//	// Convert and validate format
//	format = squiresvc.OutputFormat(*requiresListOpts.Format)
//	if !format.IsValid() {
//		err = NewErr(ErrCommand, ErrRequires, "error", "invalid format", "format", *requiresListOpts.Format)
//		goto end
//	}
//
//	// Default to current directory if none provided
//	if *requiresListOpts.DirArg == "" {
//		dir = "."
//	} else {
//		dir = *requiresListOpts.DirArg
//	}
//
//	// Discover modules
//	ms, err = squiresvc.DiscoverModules(dir)
//	if err != nil {
//		err = NewErr(ErrCommand, ErrRequires, "error", "failed to discover modules", err)
//		goto end
//	}
//
//	// Order modules
//	ordered, err = ms.OrderModules()
//	if err != nil {
//		err = NewErr(ErrCommand, ErrRequires, "error", "failed to order modules", err)
//		goto end
//	}
//
//	// Output in requested format
//	err = c.outputModules(ordered, format, config)
//	if err != nil {
//		goto end
//	}
//
//end:
//	return err
//}
//
//// outputModules outputs modules in the requested format
//func (c *RequiresListCmd) outputModules(modules squiresvc.Modules, format squiresvc.OutputFormat, config *squiresvc.Config) (err error) {
//	var buf bytes.Buffer
//
//	switch format {
//	case squiresvc.JSONOutputFormat:
//		config.Writer.Printf("%s\n", modules.JSON())
//	case squiresvc.CSVOutputFormat:
//		err = modules.CSV(&buf)
//		if err != nil {
//			err = NewErr(ErrCommand, ErrRequires, "format", "csv", err)
//			goto end
//		}
//		config.Writer.Printf("%s", buf.String())
//	case squiresvc.TableOutputFormat:
//		config.Writer.Printf("%s\n", modules.TableWriter().Render())
//	default:
//		err = NewErr(ErrCommand, ErrRequires, "error", "invalid format", "format", format)
//		goto end
//	}
//
//end:
//	return err
//}
