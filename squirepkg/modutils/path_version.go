package modutils

import (
	"fmt"

	"github.com/mikeschinkel/go-dt"
)

type PathVersion struct {
	Path    ModulePath
	Version dt.Version
}

func NewPathVersion(path ModulePath, version dt.Version) PathVersion {
	return PathVersion{
		Path:    path,
		Version: version,
	}
}
func (pv PathVersion) String() string {
	return fmt.Sprintf("%s@%s", pv.Path, pv.Version)
}
func (pv PathVersion) PathAtVersion() string {
	return fmt.Sprintf("%s@%s", pv.Path, pv.Version)
}
func (pv PathVersion) PathAt() string {
	return fmt.Sprintf("%s@", pv.Path)
}
