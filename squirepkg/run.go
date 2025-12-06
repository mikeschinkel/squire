package squirepkg

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-logutil"
	"github.com/mikeschinkel/squire/squirepkg/squire"

	_ "github.com/mikeschinkel/squire/squirepkg/squirecmds"
)

// Run starts the xmlui-test-server with the provided configuration and context.
// This is the main server execution function that initializes all components
// and starts the HTTP server.
//
// The function follows this execution flow:
//  1. Initialize global settings (writer, logger)
//  2. ParseBytes and validate options
//  3. Initialize database connection
//  4. Load API configuration
//  5. Create and configure server
//  6. Start HTTP server and listen for requests
//
// Returns ErrServerError if the server terminates with an error condition.
func Run(ctx Context, args *RunArgs) (err error) {
	var runner *cliutil.CmdRunner
	var cmd cliutil.Command
	//var opts *squire.Options
	//
	//rawOpts := args.Options

	args.Logger.Info("%s starting\n", args.AppInfo.Name())

	err = Initialize(args)
	if err != nil {
		goto end
	}

	// Set up command runner
	runner = cliutil.NewCmdRunner(cliutil.CmdRunnerArgs{
		Context: ctx,
		Args:    args.CLIArgs,
		AppInfo: args.AppInfo,
		Logger:  args.Logger,
		Writer:  args.Writer,
		Config:  args.Config,
		Options: args.Options,
	})

	// Parse the gcommand and the global and command-specific flags
	cmd, err = runner.ParseCmd(args.CLIArgs)
	if err != nil {
		goto end
	}

	// Execute command
	err = runner.RunCmd(cmd)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			cliutil.Printf("Command failed: %v", err)
			logger.Error("Run aborted", "error", err)
			os.Exit(1)
		}
		cliutil.Printf("Operation cancelled by user")
	}

end:
	return err
}

// Initialize sets up the squire package with the provided options
// TODO This code is a mess. Needs to be streamlined and cleaned up
func Initialize(args *RunArgs) (err error) {

	if args == nil {
		args = &RunArgs{}
	}

	cliutil.SetWriter(args.Writer)

	// Setting the logger sets the package level logger variable so it is accessible
	// throughout the package.
	squire.SetLogger(args.Logger)

	if args.Logger == nil {
		err = errors.New("squirepkg.Initialize: Logger not set")
		goto end
	}

	err = logutil.CallInitializerFuncs(logutil.InitializerArgs{
		AppInfo: args.AppInfo,
		Logger:  args.Logger,
	})

	if err != nil {
		err = fmt.Errorf("squirepkg.Initialize: %w", err)
		goto end
	}
	initializeLoggers(args.Logger)

	if args.Writer == nil {
		err = errors.New("squirepkg.Initialize: Writer not set")
		goto end
	}
	err = cliutil.Initialize(args.Writer)
	if args.Writer == nil {
		err = fmt.Errorf("squirepkg.Initialize: clituil.Initialize() failed; %w", err)
		goto end
	}

	// Setting the writer allows the shorthand of being able to call cliutil.Printf()
	// and cliutil.Errorf() without having a writer injected into every func.
	initializeWriters(args.Writer)
	err = cliutil.CallInitializerFuncs(cliutil.InitializerArgs{
		Writer: args.Writer,
	})

end:
	return err
}

func initializeWriters(w CLIWriter) {

	// This is redundant as it was already done in cliutil.Initialize(), but leaving
	// it here for symmetry.
	cliutil.SetWriter(w)
}

func initializeLoggers(logger *slog.Logger) {
	//TODO These should be registered by the package, not hard-coded here.
	// OR if possible resolved via reflection
	// cliutil no longer uses a logger, only a Writer
	cfgstore.SetLogger(logger)
}
