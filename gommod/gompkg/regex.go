package gompkg

import (
	"regexp"
)

var unnecessaryDirsRegex = regexp.MustCompile(`(\.git|node_modules)$`)
