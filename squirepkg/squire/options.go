package squire

import (
	"github.com/mikeschinkel/go-cliutil"
)

var _ cliutil.Options = (*Options)(nil)
var _ cliutil.GlobalOptionsGetter = (*Options)(nil)

// Options contains parsed, strongly-typed runtime options for the Squire CLI.
// It embeds *cliutil.GlobalOptions to inherit standard CLI behaviors like Quiet() and Verbosity().
type globalOptions = cliutil.GlobalOptions
type Options struct {
	*globalOptions
	// TODO: Add CLI-specific typed fields here as needed
}

func (opts *Options) Options() {}

func (opts *Options) GlobalOptions() *cliutil.GlobalOptions {
	return opts.globalOptions
}

func NewOptions(args OptionsArgs) *Options {
	return &Options{globalOptions: args.GlobalOptions}
}

type OptionsArgs struct {
	GlobalOptions *cliutil.GlobalOptions
	// TODO: Add CLI-specific typed fields here as needed
}
