package gitutils

import (
	"fmt"
)

type UpstreamState struct {
	ahead  *int
	behind *int
}

func NewUpstreamState(ahead int, behind int) *UpstreamState {
	return &UpstreamState{
		ahead:  &ahead,
		behind: &behind,
	}
}

func (us UpstreamState) UpToDate() (utd bool) {
	if !us.initialized() {
		goto end
	}
	if *us.ahead != 0 {
		goto end
	}
	if *us.behind != 0 {
		goto end
	}
	utd = true
end:
	return utd
}
func (us UpstreamState) initialized() (initialized bool) {
	if us.ahead == nil {
		goto end
	}
	if us.behind == nil {
		goto end
	}
	initialized = true
end:
	return initialized
}

func (us UpstreamState) String() (s string) {
	switch us.initialized() {
	case true:
		s = fmt.Sprintf("ahead %d / behind %d", *us.ahead, *us.behind)
	default:
		s = "unknown (no upstream configured)"
	}
	return s
}

func (us UpstreamState) Ahead() (ahead int) {
	if us.ahead != nil {
		ahead = *us.ahead
	}
	return ahead
}
func (us UpstreamState) Behind() (behind int) {
	if us.behind != nil {
		behind = *us.behind
	}
	return behind
}
