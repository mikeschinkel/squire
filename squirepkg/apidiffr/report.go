package apidiffr

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

type PackageChanges struct {
	ImportPath    dt.DirPath
	Breaking      []string
	NonBreaking   []string
	Informational []string
}

type Report struct {
	LoadErrors error
	Packages   []PackageChanges
}

func (r Report) HasBreakingChanges() bool {
	for _, p := range r.Packages {
		if len(p.Breaking) > 0 {
			return true
		}
	}
	return false
}

func (r Report) WritePackageChanges(w io.Writer) {
	if r.LoadErrors != nil {
		r.write(w, "- Package load issues:")
		for _, e := range Errors(r.LoadErrors) {
			r.write(w, "  - %s\n", e)
		}
	}

	if len(r.Packages) == 0 {
		r.write(w, "- No packages analyzed.")
		goto end
	}

	sort.Slice(r.Packages, func(i, j int) bool {
		return r.Packages[i].ImportPath < r.Packages[j].ImportPath
	})

	for _, pkg := range r.Packages {
		r.write(w, "- %s\n", pkg.ImportPath)
		r.writeList(w, "Breaking", pkg.Breaking)
		r.writeList(w, "Non-breaking", pkg.NonBreaking)
		r.writeList(w, "Informational", pkg.Informational)
	}

end:
	return
}

func (r Report) write(w io.Writer, msgs ...any) {
	switch len(msgs) {
	case 0:
	case 1:
		_, _ = fmt.Fprint(w, msgs[0].(string))
	default:
		_, _ = fmt.Fprintf(w, msgs[0].(string), msgs[1:]...)
	}
}

func (r Report) writeList(w io.Writer, label string, items []string) {
	if len(items) == 0 {
		return
	}
	sort.Strings(items)
	r.write(w, "  - %s:\n", label)
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		r.write(w, "    - %s\n", item)
	}
}
