package squirepkg

import (
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/squire/squirepkg/squire"
	"github.com/mikeschinkel/squire/squirepkg/squirecfg"
)

// ParseOptions converts raw options from clicfg.Options into
// validated squire.Options with embedded cliutil.GlobalOptions.
// If cliutilOpts is provided, it will be used directly instead of creating a new GlobalOptions.
func ParseOptions(cfgOpts *squirecfg.Options, cliutilOpts *cliutil.GlobalOptions) (opts *squire.Options, err error) {
	var globalOpts *cliutil.GlobalOptions

	// If cliutilOpts provided, use it directly (preserves all flags like --force, --dry-run)
	if cliutilOpts != nil {
		globalOpts = cliutilOpts
	} else {
		// Otherwise create new GlobalOptions from config values
		globalOpts, err = cliutil.NewGlobalOptions(cliutil.GlobalOptionsArgs{
			Quiet:     &cfgOpts.Quiet,
			Verbosity: &cfgOpts.Verbosity,
		})
		if err != nil {
			goto end
		}
	}

	opts = squire.NewOptions(squire.OptionsArgs{
		GlobalOptions: globalOpts,
	})

end:
	return opts, err
}
