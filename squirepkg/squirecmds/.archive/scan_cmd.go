package _archive

//import (
//	"github.com/mikeschinkel/go-cliutil"
//	"github.com/mikeschinkel/go-dt"
//	"github.com/mikeschinkel/go-dt/dtx"
//	"github.com/mikeschinkel/squire/squirepkg/retinue"
//)
//
//var scanOpts = &struct {
//	Directory     *string
//	ContinueOnErr *bool
//	SilenceErrs   *bool
//}{
//	Directory:     new(string),
//	ContinueOnErr: new(bool),
//	SilenceErrs:   new(bool),
//}
//
//var ScanFlagSet = &cliutil.FlagSet{
//	Name: "scan",
//	FlagDefs: []cliutil.FlagDef{
//		{
//			Name:     "continue",
//			Usage:    "On error, accumulate and continue; do not fail fast",
//			Required: false,
//			Default:  false,
//			Bool:     scanOpts.ContinueOnErr,
//		},
//		{
//			Name:     "silence-errors",
//			Usage:    "Suppress error messages to stderr (permission denied, etc.)",
//			Required: false,
//			Default:  false,
//			Bool:     scanOpts.SilenceErrs,
//		},
//	},
//}
//
//var _ cliutil.CommandHandler = (*ScanCmd)(nil)
//
//// ScanCmd discovers Go modules under specified roots
//type ScanCmd struct {
//	*cliutil.CmdBase
//}
//
//func init() {
//	err := cliutil.RegisterCommand(&ScanCmd{
//		CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
//			Order:       10,
//			Name:        "scan",
//			Usage:       "scan [<dir>]",
//			Description: "Discover Go modules in unmanaged repos (defaults to current directory)",
//			FlagSets:    []*cliutil.FlagSet{ScanFlagSet},
//			ArgDefs: []*cliutil.ArgDef{
//				{
//					Name:     "dir",
//					Usage:    "Directory to scan (defaults to current directory)",
//					Required: false,
//					Default:  "",
//					String:   scanOpts.Directory,
//					Example:  "~/Projects",
//				},
//			},
//		}),
//	})
//	if err != nil {
//		panic(err)
//	}
//}
//
//// Handle executes the scan command
//func (c *ScanCmd) Handle() (err error) {
//	var dp dt.DirPath
//
//	dp, err = dt.ParseDirPath(*scanOpts.Directory)
//	if err != nil {
//		goto end
//	}
//	_, err = retinue.FindGoModFiles[dt.DirPath](retinue.FindGoModFilesArgs{
//		DirPaths:      []dt.DirPath{dp},
//		ContinueOnErr: *scanOpts.ContinueOnErr,
//		Logger:        c.Logger,
//		Writer:        c.Writer,
//		SilenceErrs:   *scanOpts.SilenceErrs,
//		SkipBehavior:  retinue.SkipManaged,
//		MatchBehavior: dtx.WriteOnMatch,
//		ParseEntryFunc: func(ep dt.EntryPath, _ dt.DirPath, _ dt.DirEntry) dt.EntryPath {
//			return dt.EntryPath(ep.Dir())
//		},
//	})
//	if err != nil && !*scanOpts.SilenceErrs {
//		c.Writer.Errorf("\nERROR: Errors occurred:\n%v", err)
//		goto end
//	}
//end:
//	return err
//}
