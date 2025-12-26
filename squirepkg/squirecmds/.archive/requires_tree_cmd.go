package _archive

//import (
//	"github.com/mikeschinkel/go-cliutil"
//	"github.com/mikeschinkel/go-dt"
//	"github.com/mikeschinkel/squire/squirepkg/squiresvc"
//)
//
//var requiresTreeOpts = &struct {
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
//var RequiresTreeFlagSet = &cliutil.FlagSet{
//	Name: "requires-tree",
//	FlagDefs: []cliutil.FlagDef{
//		{
//			Name:     "all",
//			Usage:    "Include external (non-Squire) modules in the tree",
//			Required: false,
//			Default:  false,
//			Bool:     requiresTreeOpts.All,
//		},
//		{
//			Name:     "show-dirs",
//			Usage:    "Show relative directory paths for local modules",
//			Required: false,
//			Default:  false,
//			Bool:     requiresTreeOpts.ShowDirs,
//		},
//		{
//			Name:     "show-paths",
//			Usage:    "Show full module paths instead of short names",
//			Required: false,
//			Default:  false,
//			Bool:     requiresTreeOpts.ShowPaths,
//		},
//		{
//			Name:     "show-all",
//			Usage:    "Show both module path and location (implies --show-dirs)",
//			Required: false,
//			Default:  false,
//			Bool:     requiresTreeOpts.ShowAll,
//		},
//		{
//			Name:     "embed",
//			Usage:    "Markdown file to embed the tree into",
//			Required: false,
//			Default:  "",
//			String:   requiresTreeOpts.EmbedFile,
//		},
//		{
//			Name:     "before",
//			Usage:    "Insert tree before the marker (requires --embed)",
//			Required: false,
//			Default:  false,
//			Bool:     requiresTreeOpts.Before,
//		},
//		{
//			Name:     "after",
//			Usage:    "Insert tree after the marker (requires --embed)",
//			Required: false,
//			Default:  false,
//			Bool:     requiresTreeOpts.After,
//		},
//	},
//}
//
//var _ cliutil.CommandHandler = (*RequiresTreeCmd)(nil)
//
//// RequiresTreeCmd visualizes module dependencies as a tree
//type RequiresTreeCmd struct {
//	*cliutil.CmdBase
//}
//
//func init() {
//	err := cliutil.RegisterCommand(&RequiresTreeCmd{
//		CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
//			Order:       21,
//			Name:        "tree",
//			Usage:       "tree [<dir>] [flags]",
//			Description: "Visualize module dependencies as a tree",
//			FlagSets:    []*cliutil.FlagSet{RequiresTreeFlagSet},
//			ArgDefs: []*cliutil.ArgDef{
//				{
//					Name:     "dir",
//					Usage:    "Directory to analyze (defaults to current directory)",
//					Required: false,
//					Default:  "",
//					String:   requiresTreeOpts.DirArg,
//					Example:  ".",
//				},
//			},
//		}),
//	}, requiresCmd)
//	if err != nil {
//		panic(err)
//	}
//}
//
//// Handle executes the requires tree command
//func (c *RequiresTreeCmd) Handle() (err error) {
//	var config *squiresvc.Config
//	var dir string
//	var ms *squiresvc.ModuleSet
//	var opts squiresvc.TreeOptions
//	var treeOutput string
//	var embedPath dt.Filepath
//
//	config = c.Config.(*squiresvc.Config)
//
//	// Validate flag combinations
//	err = validateFlags()
//	if err != nil {
//		goto end
//	}
//
//	// Default to current directory if none provided
//	if *requiresTreeOpts.DirArg == "" {
//		dir = "."
//	} else {
//		dir = *requiresTreeOpts.DirArg
//	}
//
//	// Discover modules
//	ms, err = squiresvc.DiscoverModules(dir)
//	if err != nil {
//		err = NewErr(ErrCommand, ErrTree, err)
//		goto end
//	}
//
//	// Build tree options
//	opts = squiresvc.TreeOptions{
//		ShowDirs:  *requiresTreeOpts.ShowDirs,
//		ShowPaths: *requiresTreeOpts.ShowPaths,
//		ShowAll:   *requiresTreeOpts.ShowAll,
//		ShowExt:   *requiresTreeOpts.All,
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
//	if *requiresTreeOpts.EmbedFile != "" {
//		// Embed into markdown file
//		embedPath = dt.Filepath(*requiresTreeOpts.EmbedFile)
//		err = squiresvc.EmbedTree(embedPath, treeOutput, *requiresTreeOpts.Before)
//		if err != nil {
//			err = NewErr(ErrCommand, ErrTree, ErrFileWrite, "file", *requiresTreeOpts.EmbedFile, err)
//			goto end
//		}
//		config.Writer.Printf("Tree embedded into %s\n", *requiresTreeOpts.EmbedFile)
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
//func validateFlags() (err error) {
//	var hasEmbed bool
//	var hasBefore bool
//	var hasAfter bool
//
//	hasEmbed = *requiresTreeOpts.EmbedFile != ""
//	hasBefore = *requiresTreeOpts.Before
//	hasAfter = *requiresTreeOpts.After
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
