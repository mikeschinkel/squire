package squirepkg

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt/appinfo"
	"github.com/mikeschinkel/squire/squirepkg/common"
	"github.com/mikeschinkel/squire/squirepkg/squirecfg"
)

// RunCLI is the main CLI entry point for the xmlui-test-server application.
// It handles command-line argument parsing, configuration loading, and starts the server.
// This function sets up logging, loads configuration files, and delegates to Run().
//
// Exit codes:
//   - 1: Options parsing failure
//   - 2: Configuration loading failure
//   - 3: Configuration parsing failure
//   - 4: Known runtime error
//   - 5: Unknown runtime error
//   - 6: Logger setup failure
func RunCLI() {
	var err error
	//var squirecfg *squirecfg.RootConfigV1
	var config *common.Config
	var squireCfg *squirecfg.RootConfigV1
	var cliOptions *cliutil.CLIOptions
	var cfgOptions *squirecfg.Options
	var options *common.Options
	var args []string
	var appInfo appinfo.AppInfo
	var wr cliutil.WriterLogger
	//
	cliOptions, args, err = cliutil.ParseCLIOptions(os.Args)
	if err != nil {
		stdErrf("Invalid option(s): %v\n", strings.Replace(err.Error(), "\n", "; ", -1))
		os.Exit(cliutil.ExitOptionsParseError)
	}
	if cliOptions == nil {
		// This should never happen as ParseCLIOptions returns an error if it fails
		panic("cliutil.ParseCLIOptions returned nil without error")
	}

	// Convert cliutil.GlobalOptions to squirecfg.Options (raw)
	quiet := cliOptions.Quiet()
	verbosity := int(cliOptions.Verbosity())
	cfgOptions = squirecfg.NewOptions(squirecfg.OptionsArgs{
		Quiet:     &quiet,
		Verbosity: &verbosity,
	})

	// Parse raw options to typed options (pass cliOptions to preserve all flags)
	options, err = ParseOptions(cfgOptions, cliOptions)
	if err != nil {
		stdErrf("Failed to parse options: %v\n", err)
		os.Exit(cliutil.ExitOptionsParseError)
	}

	appInfo = AppInfo()
	wr, err = cfgstore.CreateWriterLogger(&cfgstore.WriterLoggerArgs{
		Quiet:      quiet,
		Verbosity:  options.Verbosity(),
		ConfigSlug: appInfo.ConfigSlug(),
		LogFile:    common.LogFile,
	})
	if err != nil {
		stdErrf("Failed to run: %v\n", err)
		os.Exit(cliutil.ExitLoggerSetupError)
	}

	squireCfg, err = squirecfg.LoadRootConfigV1(squirecfg.LoadRootConfigV1Args{
		AppInfo: appInfo,
		Options: cfgOptions,
	})
	if err != nil {
		wr.Writer.Errorf("Failed to load config file(s); %v\n", err)
		os.Exit(cliutil.ExitConfigLoadError)
	}

	config, err = ParseConfig(squireCfg, common.ConfigArgs{
		Options: options,
		AppInfo: appInfo,
		Writer:  wr.Writer,
		Logger:  wr.Logger,
	})

	if err != nil {
		wr.Writer.Errorf("Failed to parse config file(s); %v\n", err)
		wr.Logger.Error("Failed to parse config file(s)", "error", err)
		os.Exit(cliutil.ExitConfigParseError)
	}

	// TODO: Make 30 second timeout configurable
	context.WithTimeout(context.Background(), 30*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = Run(ctx, &RunArgs{
		AppInfo: appInfo,
		Config:  config,
		Writer:  wr.Writer,
		Logger:  wr.Logger,
		Options: options,
		CLIArgs: args,
	})

	if err != nil {
		wr.Writer.Errorf("Error running %s: %v", common.AppName, err)
		wr.Logger.Error("Unknown runtime error", "error", err)
		os.Exit(cliutil.ExitUnknownRuntimeError)
	}
}
