package gitutils

// Staging indicates where a change exists (index vs worktree).
// IndexStaging and WorktreeStaging values match git status --porcelain positions: 0=index, 1=worktree.
// NoneStaging and BothStaging are logical states, not position indices.
type Staging byte

const (
	NoneStaging     Staging = 255 // No change (use 255 to avoid conflict with 0/1)
	IndexStaging    Staging = 0   // Staged (position 0 in git status)
	WorktreeStaging Staging = 1   // Unstaged (position 1 in git status)
	BothStaging     Staging = 2   // Both staged and unstaged (e.g., "MM" in git status)
)

// String returns the full name of the staging state for general purpose use.
func (s Staging) String() string {
	switch s {
	case IndexStaging:
		return "Staged"
	case WorktreeStaging:
		return "Unstaged"
	case BothStaging:
		return "Both Staged and Unstaged"
	case NoneStaging:
		return "None"
	default:
		return "Unknown"
	}
}

// Label returns a short display label for the staging state (suitable for table columns).
func (s Staging) Label() string {
	switch s {
	case IndexStaging:
		return "Staged"
	case WorktreeStaging:
		return "Unstaged"
	case BothStaging:
		return "Both"
	case NoneStaging:
		return "---"
	default:
		return "???"
	}
}

// ParseStaging determines staging state from staged and unstaged change types.
// Returns IndexStaging if only staged change exists.
// Returns WorktreeStaging if only unstaged change exists.
// Returns BothStaging if both staged and unstaged changes exist (e.g., file was staged then modified again).
// Returns NoneStaging if neither change exists.
func ParseStaging(stagedChange, unstagedChange ChangeType) (s Staging, err error) {
	hasStaged := stagedChange != UnknownChangeType
	hasUnstaged := unstagedChange != UnknownChangeType

	switch {
	case hasStaged && hasUnstaged:
		s = BothStaging
	case hasStaged:
		s = IndexStaging
	case hasUnstaged:
		s = WorktreeStaging
	default:
		s = NoneStaging
	}

	return s, nil
}
