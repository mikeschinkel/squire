package gomtui

import (
	"strings"

	"github.com/mikeschinkel/go-dt"
)

type DispositionsCache struct {
	cache map[dt.RelFilepath]FileDisposition
}

func (dc DispositionsCache) LabelsMap() map[dt.RelFilepath]dt.Identifier {
	labelsMap := make(map[dt.RelFilepath]dt.Identifier, len(dc.cache))
	for path, disp := range dc.cache {
		// Store as lowercase label (e.g., "commit", "omit", "ignore", "exclude")
		labelsMap[path] = dt.Identifier(strings.ToLower(disp.Label()))
	}
	return labelsMap
}

func NewDispositionsCache(fs *FileSource) (dc *DispositionsCache) {
	dc = &DispositionsCache{
		cache: make(map[dt.RelFilepath]FileDisposition),
	}
	for _, file := range fs.Files() {
		_, exists := dc.cache[file.Path]
		if exists {
			continue
		}
		// Only set UnspecifiedDisposition if not already loaded from saved plan
		dc.cache[file.Path] = UnspecifiedDisposition
	}
	return dc
}
