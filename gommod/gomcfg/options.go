package gomcfg

import (
	"github.com/mikeschinkel/go-cliutil"
)

const (
	DefaultQuiet     = false
	DefaultVerbosity = cliutil.DefaultVerbosity
)

type Options struct {
	Quiet     bool
	Verbosity int
}

func (*Options) Options() {}

type OptionsArgs struct {
	Quiet     *bool
	Verbosity *int
}

func NewOptions(args OptionsArgs) *Options {
	opts := &Options{}

	if args.Quiet != nil {
		opts.Quiet = *args.Quiet
	} else {
		opts.Quiet = DefaultQuiet
	}

	if args.Verbosity != nil {
		opts.Verbosity = *args.Verbosity
	} else {
		opts.Verbosity = DefaultVerbosity
	}

	return opts
}
