package _archive

//
//import (
//	"github.com/mikeschinkel/go-cliutil"
//	"github.com/mikeschinkel/go-dt"
//	"github.com/mikeschinkel/gomion/gommod/gompkg"
//)
//
//var treeOpts = &struct {
//	DirArg    *string
//	ShowDirs  *bool
//	ShowPaths *bool
//	ShowAll   *bool
//	All       *bool
//	EmbedFile *string
//	Before    *bool
//	After     *bool
//}{
//	DirArg:    new(string),
//	ShowDirs:  new(bool),
//	ShowPaths: new(bool),
//	ShowAll:   new(bool),
//	All:       new(bool),
//	EmbedFile: new(string),
//	Before:    new(bool),
//	After:     new(bool),
//}
//
//var treeFlagSet = &cliutil.FlagSet{
//	Name: "tree",
//	FlagDefs: []cliutil.FlagDef{
//		{
//			Name:     "all",
//			Usage:    "Include external (non-Gomion) modules in the tree",
//			Required: false,
//			Default:  false,
//			Bool:     treeOpts.All,
//		},
//		{
//			Name:     "show-dirs",
//			Usage:    "Show relative directory paths for local modules",
//			Required: false,
//			Default:  false,
//			Bool:     treeOpts.ShowDirs,
//		},
//		{
//			Name:     "show-paths",
//			Usage:    "Show full module paths instead of short names",
//			Required: false,
//			Default:  false,
//			Bool:     treeOpts.ShowPaths,
//		},
//		{
//			Name:     "show-all",
//			Usage:    "Show both module path and location (implies --show-dirs)",
//			Required: false,
//			Default:  false,
//			Bool:     treeOpts.ShowAll,
//		},
//		{
//			Name:     "embed",
//			Usage:    "Markdown file to embed the tree into",
//			Required: false,
//			Default:  "",
//			String:   treeOpts.EmbedFile,
//		},
//		{
//			Name:     "before",
//			Usage:    "Insert tree before the marker (requires --embed)",
//			Required: false,
//			Default:  false,
//			Bool:     treeOpts.Before,
//		},
//		{
//			Name:     "after",
//			Usage:    "Insert tree after the marker (requires --embed)",
//			Required: false,
//			Default:  false,
//			Bool:     treeOpts.After,
//		},
//	},
//}
//
//var _ cliutil.CommandHandler = (*treeCmd)(nil)
//
//// treeCmd visualizes module dependencies as a tree
//type treeCmd struct {
//	*cliutil.CmdBase
//}
//
//func init() {
//	err := cliutil.RegisterCommand(&treeCmd{
//		CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
//			Order:       21,
//			Name:        "tree",
//			Usage:       "tree [<dir>] [flags]",
//			Description: "Visualize module dependencies as a tree",
//			FlagSets:    []*cliutil.FlagSet{treeFlagSet},
//			ArgDefs: []*cliutil.ArgDef{
//				{
//					Name:     "dir",
//					Usage:    "Directory to analyze (defaults to current directory)",
//					Required: false,
//					Default:  "",
//					String:   treeOpts.DirArg,
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
//// Handle executes the requires tree command
//func (c *treeCmd) Handle() (err error) {
//	var config *gompkg.Config
//	var dir string
//	var ms *gompkg.ModuleSet
//	var opts gompkg.TreeOptions
//	var treeOutput string
//	var embedPath dt.Filepath
//
//	config = c.Config.(*gompkg.Config)
//
//	// Validate flag combinations
//	err = validateTreeFlags()
//	if err != nil {
//		goto end
//	}
//
//	// Default to current directory if none provided
//	if *treeOpts.DirArg == "" {
//		dir = "."
//	} else {
//		dir = *treeOpts.DirArg
//	}
//
//	// Discover modules
//	ms, err = gompkg.DiscoverModules(dir)
//	if err != nil {
//		err = NewErr(ErrCommand, ErrTree, err)
//		goto end
//	}
//
//	// Build tree options
//	opts = gompkg.TreeOptions{
//		ShowDirs:  *treeOpts.ShowDirs,
//		ShowPaths: *treeOpts.ShowPaths,
//		ShowAll:   *treeOpts.ShowAll,
//		ShowExt:   *treeOpts.All,
//	}
//
//	// Render tree
//	treeOutput, err = ms.RenderTree(opts)
//	if err != nil {
//		err = NewErr(ErrCommand, ErrTree, err)
//		goto end
//	}
//
//	// Handle embedding or stdout
//	if *treeOpts.EmbedFile != "" {
//		// Embed into markdown file
//		embedPath = dt.Filepath(*treeOpts.EmbedFile)
//		err = gompkg.EmbedTree(embedPath, treeOutput, *treeOpts.Before)
//		if err != nil {
//			err = NewErr(ErrCommand, ErrTree, ErrFileWrite, "file", *treeOpts.EmbedFile, err)
//			goto end
//		}
//		config.Writer.Printf("Tree embedded into %s\n", *treeOpts.EmbedFile)
//	} else {
//		// Print to stdout
//		config.Writer.Printf("%s", treeOutput)
//	}
//
//end:
//	return err
//}
//
//// validateFlags validates the flag combinations
//func validateTreeFlags() (err error) {
//	var hasEmbed bool
//	var hasBefore bool
//	var hasAfter bool
//
//	hasEmbed = *treeOpts.EmbedFile != ""
//	hasBefore = *treeOpts.Before
//	hasAfter = *treeOpts.After
//
//	// If --embed is not specified, --before and --after are invalid
//	if !hasEmbed && (hasBefore || hasAfter) {
//		err = NewErr(ErrCommand, ErrTree, ErrInvalidFlags, "error", "--before and --after require --embed")
//		goto end
//	}
//
//	// If --embed is specified, exactly one of --before or --after is required
//	if hasEmbed {
//		if !hasBefore && !hasAfter {
//			err = NewErr(ErrCommand, ErrTree, ErrInvalidFlags, "error", "--embed requires either --before or --after")
//			goto end
//		}
//
//		if hasBefore && hasAfter {
//			err = NewErr(ErrCommand, ErrTree, ErrInvalidFlags, "error", "--before and --after are mutually exclusive")
//			goto end
//		}
//	}
//
//end:
//	return err
//}
