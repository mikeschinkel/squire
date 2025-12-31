package gommod

import (
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt/appinfo"
)

// RunArgs contains all the configuration and dependencies needed to run the server.
// This struct is used to pass configuration from the CLI layer to the core server logic.
type RunArgs struct {
	CLIArgs []string        // Should have os.Args[1:] (omits program name)
	AppInfo appinfo.AppInfo // "Static" Application info
	Options cliutil.Options //
	Config  cliutil.Config  // Loaded configuration from files
	Writer  cliutil.Writer  // Writer for CLI output and logging
	Logger  *slog.Logger    // Structured logger instance
}
