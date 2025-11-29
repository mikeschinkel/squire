package common

import (
	"github.com/mikeschinkel/go-cliutil"
)

var _ cliutil.Options = (*Options)(nil)
var _ cliutil.CLIOptionsGetter = (*Options)(nil)

// Options contains parsed, strongly-typed runtime options for the XMLUI CLI.
// It embeds *cliutil.GlobalOptions to inherit standard CLI behaviors like Quiet() and Verbosity().
type cliOptions = cliutil.CLIOptions
type Options struct {
	*cliOptions
	// TODO: Add CLI-specific typed fields here as needed
}

func (opts Options) CLIOptions() *cliutil.CLIOptions {
	return opts.cliOptions
}

func NewOptions(args OptionsArgs) *Options {
	return &Options{cliOptions: args.CLIOptions}
}

type OptionsArgs struct {
	CLIOptions *cliutil.CLIOptions
	// TODO: Add CLI-specific typed fields here as needed
}
