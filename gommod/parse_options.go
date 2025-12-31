package gommod

import (
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/gomion/gommod/gomcfg"
	"github.com/mikeschinkel/gomion/gommod/gomion"
)

// ParseOptions converts raw options from clicfg.Options into
// validated gomion.Options with embedded cliutil.GlobalOptions.
// If cliutilOpts is provided, it will be used directly instead of creating a new GlobalOptions.
func ParseOptions(cfgOpts *gomcfg.Options, cliutilOpts *cliutil.GlobalOptions) (opts *gomion.Options, err error) {
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

	opts = gomion.NewOptions(gomion.OptionsArgs{
		GlobalOptions: globalOpts,
	})

end:
	return opts, err
}
