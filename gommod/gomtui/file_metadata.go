package gomtui

import (
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/bubbletree"
)

// batchLoadMeta loads metadata for multiple files in batch.
// Calls file.LoadMeta() for each file and collects errors.
// Returns error if any file failed to load metadata.
func batchLoadMeta(files []*bubbletree.File, root dt.DirPath) (err error) {
	var errs []error

	for _, file := range files {
		errs = AppendErr(errs, file.LoadMeta(root))
	}

	if len(errs) > 0 {
		// Combine all errors
		err = CombineErrs(errs)
	}

	return err
}
