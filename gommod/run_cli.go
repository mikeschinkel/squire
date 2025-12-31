package gommod

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt/appinfo"
	"github.com/mikeschinkel/gomion/gommod/gomcfg"
	"github.com/mikeschinkel/gomion/gommod/gomion"
	"github.com/mikeschinkel/gomion/gommod/gompkg"
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
	//var gomcfg *gomcfg.RootConfigV1
	var config *gompkg.Config
	var gomionCfg *gomcfg.RootConfigV1
	var globalOptions *cliutil.GlobalOptions
	var cfgOptions *gomcfg.Options
	var options *gomion.Options
	var args []string
	var appInfo appinfo.AppInfo
	var wr cliutil.WriterLogger
	//
	globalOptions, args, err = cliutil.ParseGlobalOptions(os.Args)
	if err != nil {
		stdErrf("Invalid option(s): %v\n", strings.Replace(err.Error(), "\n", "; ", -1))
		os.Exit(cliutil.ExitOptionsParseError)
	}
	if globalOptions == nil {
		// This should never happen as ParseGlobalOptions returns an error if it fails
		panic("cliutil.ParseGlobalOptions returned nil without error")
	}

	// Convert cliutil.GlobalOptions to gomcfg.Options (raw)
	quiet := globalOptions.Quiet()
	verbosity := int(globalOptions.Verbosity())
	cfgOptions = gomcfg.NewOptions(gomcfg.OptionsArgs{
		Quiet:     &quiet,
		Verbosity: &verbosity,
	})

	// Parse raw options to typed options (pass globalOptions to preserve all flags)
	options, err = ParseOptions(cfgOptions, globalOptions)
	if err != nil {
		stdErrf("Failed to parse options: %v\n", err)
		os.Exit(cliutil.ExitOptionsParseError)
	}

	appInfo = AppInfo()
	wr, err = cfgstore.CreateWriterLogger(&cfgstore.WriterLoggerArgs{
		Quiet:      quiet,
		Verbosity:  options.Verbosity(),
		ConfigSlug: appInfo.ConfigSlug(),
		LogFile:    gomion.LogFile,
	})
	if err != nil {
		stdErrf("Failed to run: %v\n", err)
		os.Exit(cliutil.ExitLoggerSetupError)
	}

	gomionCfg, err = gomcfg.LoadRootConfigV1(gomcfg.LoadRootConfigV1Args{
		AppInfo: appInfo,
		Options: cfgOptions,
	})
	if err != nil {
		wr.Writer.Errorf("Failed to load config file(s); %v\n", err)
		os.Exit(cliutil.ExitConfigLoadError)
	}

	config, err = ParseConfig(gomionCfg, gompkg.ConfigArgs{
		Options: options,
		Writer:  wr.Writer,
		Logger:  wr.Logger,
	})

	if err != nil {
		wr.Writer.Errorf("Failed to parse config file(s); %v\n", err)
		wr.Logger.Error("Failed to parse config file(s)", "error", err)
		os.Exit(cliutil.ExitConfigParseError)
	}

	// TODO: Make 30 second timeout configurable
	context.WithTimeout(context.Background(), 30*time.Minute)
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
		wr.Writer.Errorf("Error running %s: %v", gomion.AppName, err)
		wr.Logger.Error("Unknown runtime error", "error", err)
		os.Exit(cliutil.ExitUnknownRuntimeError)
	}
}
