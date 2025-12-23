package modutils

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"
)

type VersionSuggestions struct {
	Compatible string
	Breaking   string
}

func SuggestNextVersions(latestTag string, breaking bool) (VersionSuggestions, error) {
	if !semver.IsValid(latestTag) {
		return VersionSuggestions{}, fmt.Errorf("invalid semver tag: %q", latestTag)
	}

	major, minor, patch, err := splitSemver(latestTag)
	if err != nil {
		return VersionSuggestions{}, err
	}

	var sugg VersionSuggestions
	if breaking {
		if major == 0 {
			sugg.Breaking = fmt.Sprintf("v%d.%d.%d", major, minor+1, 0)
		} else {
			sugg.Breaking = fmt.Sprintf("v%d.%d.%d", major+1, 0, 0)
		}
		return sugg, nil
	}

	sugg.Compatible = fmt.Sprintf("v%d.%d.%d", major, minor, patch+1)
	if major == 0 {
		sugg.Breaking = fmt.Sprintf("v%d.%d.%d", major, minor+1, 0)
	} else {
		sugg.Breaking = fmt.Sprintf("v%d.%d.%d", major+1, 0, 0)
	}
	return sugg, nil
}

func splitSemver(v string) (major, minor, patch int, err error) {
	v = strings.TrimPrefix(v, "v")
	core, _, _ := strings.Cut(v, "-")
	parts := strings.Split(core, ".")
	if len(parts) < 3 {
		return 0, 0, 0, fmt.Errorf("invalid version core: %q", v)
	}
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, err
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, err
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, err
	}
	return major, minor, patch, nil
}
