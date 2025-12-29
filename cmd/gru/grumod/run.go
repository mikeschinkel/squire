package grumod

import (
	"log/slog"
	"strings"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-logutil"
	"github.com/mikeschinkel/squire/gru/grumod/gru"
	"github.com/mikeschinkel/squire/gru/grumod/grucfg"
)

func showError(msg string, err error) {
	errMsg := err.Error()
	cliutil.Stderrf("%s: %v\n", msg, strings.ReplaceAll(errMsg, "\n", "; "))
}

func Initialize() (err error) {
	err = dt.EnsureUserHomeDir()
	return err
}

// Run executes the GRU CLI workflow and returns an exit code.
func Run() (result int) {
	var logger *slog.Logger
	var opts cliutil.Options
	var args []string

	err := Initialize()
	if err != nil {
		showError("Failed initialization", err)
		return cliutil.ExitInitializeationError
	}

	opts, args, err = cliutil.ParseOptions(grucfg.NewFlags(), NewOptions())
	if err != nil {
		showError("Invalid option(s)", err)
		return cliutil.ExitOptionsParseError
	}

	writer := cliutil.NewWriter(&cliutil.WriterArgs{
		Quiet:     opts.Quiet(),
		Verbosity: opts.Verbosity(),
	})

	logFile, err := cfgstore.ProjectConfigFilepath(gru.ConfigSlug, gru.LogFile, nil)
	switch {
	case err != nil:
		writer.Errorf("Error generating log filepath; logging disabled: %v", err)
		logger = logutil.NullLogger
	default:
		logger, err = logutil.CreateJSONFileLogger(logFile)
	}

	if err != nil {
		writer.Errorf("Error generating log to %s; logging disabled: %v", logFile, err)
		logger = logutil.NullLogger
	}
	app := cliutil.NewApp(cliutil.AppArgs{
		Writer:  writer,
		Logger:  logger,
		Options: opts,
		RunFunc: func(args []string) error {
			println("Hello world")
			return nil
		},
	})

	err = app.Run(args)

	if err != nil {
		showError("Run error", err)
		return cliutil.ExitExecutionError
	}
	return cliutil.ExitSuccess
}
