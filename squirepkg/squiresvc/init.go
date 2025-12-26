package squiresvc

import (
	"bytes"
	"fmt"
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
)

// InitReposResult contains the results of repository initialization
type InitReposResult struct {
	Initialized int
	Skipped     int
	Errors      []error
}

type InitArgs struct {
	FilePath string
	DirPath  string
	Config   *Config
	Logger   *slog.Logger
	Writer   cliutil.Writer
}

// Init handles the init command
func Init(args InitArgs) (result *InitReposResult, err error) {
	var goModFiles []dt.Filepath
	var fp dt.Filepath
	var dp dt.DirPath
	//var graph *GoModGraph
	//var config *Config

	switch {
	// Check for mutual exclusivity
	case args.FilePath != "" && args.DirPath != "":
		err = fmt.Errorf("cannot specify both <dir> and --file")
		if err != nil {
			goto end
		}
	case args.FilePath != "":
		fp, err = dt.ParseFilepath(args.FilePath)
		if err != nil {
			goto end
		}
	case args.DirPath != "":
		dp, err = dt.ParseDirPath(args.DirPath)
		if err != nil {
			goto end
		}
	}

	switch {
	case fp != "":
		goModFiles, err = loadInitFile(fp)
		if err != nil {
			goto end
		}
	case dp == "":
		dp, err = dt.ParseDirPath(".")
		if err != nil {
			goto end
		}
		fallthrough
	default: // dp != ""
		goModFiles, err = FindGoModFiles[dt.Filepath](FindGoModFilesArgs{
			DirPaths:       []dt.DirPath{dp},
			ContinueOnErr:  true,
			SilenceErrs:    false,
			SkipBehavior:   SkipUnmanaged,
			MatchBehavior:  dtx.WriteOnMatch,
			Config:         args.Config,
			Logger:         args.Logger,
			Writer:         args.Writer,
			ParseEntryFunc: nil,
		})
		if err != nil {
			goto end
		}
	}
	fmt.Printf("%#v", goModFiles)

	// TODO: Initialize
	//config = args.Config.(*Config)
	//graph = NewGoModuleGraph(goModFiles, GoModuleGraphArgs{
	//	ModuleSpecs: config.ModuleSpecs,
	//})
	//err = graph.Load()
	//fmt.Printf("%#v", graph)
	//result = &InitReposResult{}
end:
	return result, err
}

// loadInitFile loads an init file which is a simple text file containing list of
// diretories that contains go.mod files, one file per file.
func loadInitFile(fp dt.Filepath) (files []dt.Filepath, err error) {
	var exists bool
	var content []byte
	var lines [][]byte

	exists, err = fp.Exists()
	if err != nil {
		goto end
	}
	if !exists {
		err = NewErr(dt.ErrFileNotExists, "filepath", fp)
	}
	content, err = fp.ReadFile()
	if err != nil {
		goto end
	}
	lines = bytes.Split(content, []byte("\n"))
	files = make([]dt.Filepath, len(lines))
	for i, line := range lines {
		files[i] = dt.Filepath(line)
	}
end:
	return files, err
}
