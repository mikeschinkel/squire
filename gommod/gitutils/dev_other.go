//go:build !aix && !darwin && !dragonfly && !freebsd && !illumos && !linux && !netbsd && !openbsd && !solaris

package gitutils

import "fmt"

func ensureSameFilesystem(_, _ string) error {
	return fmt.Errorf("cache hardlink requirement is not supported on this platform")
}
