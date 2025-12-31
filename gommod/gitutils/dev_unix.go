//go:build aix || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || solaris

package gitutils

import (
	"fmt"
	"os"
	"syscall"

	"github.com/mikeschinkel/go-dt"
)

func ensureSameFilesystem(sourceRepoRoot, cacheBase dt.DirPath) (err error) {
	var sourceInfo, cacheInfo os.FileInfo
	var sourceStat, cacheStat *syscall.Stat_t
	var ok bool

	sourceInfo, err = sourceRepoRoot.Stat()
	if err != nil {
		goto end
	}

	cacheInfo, err = cacheBase.Stat()
	if err != nil {
		if !os.IsNotExist(err) {
			goto end
		}
		err = cacheBase.MkdirAll(0o755)
		if err != nil {
			goto end
		}
		cacheInfo, err = cacheBase.Stat()
		if err != nil {
			goto end
		}
	}

	sourceStat, ok = sourceInfo.Sys().(*syscall.Stat_t)
	if !ok || sourceStat == nil {
		err = fmt.Errorf("cannot determine filesystem device for source repo")
		goto end
	}

	cacheStat, ok = cacheInfo.Sys().(*syscall.Stat_t)
	if !ok || cacheStat == nil {
		err = fmt.Errorf("cannot determine filesystem device for cache directory")
		goto end
	}

	if sourceStat.Dev != cacheStat.Dev {
		err = fmt.Errorf("cache directory must be on the same filesystem as the repo for hardlinks (repo=%s cache=%s)", sourceRepoRoot, cacheBase)
		goto end
	}

end:
	return err
}
