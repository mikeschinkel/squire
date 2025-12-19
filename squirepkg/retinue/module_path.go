package retinue

import (
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// ModulePath is the Go module path from this module's go.mod "module" directive,
// e.g. "github.com/mikeschinkel/go-dt".
type ModulePath string
type ModulePaths []ModulePath

func (mp ModulePath) Split(sep string) (uss []dt.URLSegment) {
	uss = make([]dt.URLSegment, 0, 3)
	for _, ps := range strings.Split(string(mp), sep) {
		uss = append(uss, dt.URLSegment(ps))
	}
	return uss
}

func (mps ModulePaths) Strings() (ss []string) {
	ss = make([]string, 0, len(mps))
	for _, mp := range mps {
		ss = append(ss, string(mp))
	}
	return ss
}

func (mps ModulePaths) Join(sep string) (s string) {
	return strings.Join(mps.Strings(), sep)
}
