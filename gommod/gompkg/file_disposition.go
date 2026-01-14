package gompkg

import (
	"fmt"
	"strings"
)

const (
	UnspecifiedDisposition FileDisposition = 0
	CommitDisposition      FileDisposition = 'c'
	OmitDisposition        FileDisposition = 'o' // Special ChangeSet - skip for takes
	GitIgnoreDisposition   FileDisposition = 'i' // Add to .gitignore
	GitExcludeDisposition  FileDisposition = 'x' // Add to .git/info/exclude
)

// FileDisposition represents how a file should be handled in commit workflow
type FileDisposition byte

func (d FileDisposition) IsValid() bool {
	return d.Key() == string(d)
}

func (d FileDisposition) String() string {
	switch d {
	case CommitDisposition:
		return "Commit"
	case OmitDisposition:
		return "Omit"
	case GitIgnoreDisposition:
		return ".gitignore"
	case GitExcludeDisposition:
		return ".git/info/exclude"
	default:
		return "Unknown"
	}
}
func (d FileDisposition) Slug() string {
	return strings.ToLower(d.Label())
}
func (d FileDisposition) Label() string {
	switch d {
	case CommitDisposition:
		return "Commit"
	case OmitDisposition:
		return "Omit"
	case GitIgnoreDisposition:
		return "Ignore"
	case GitExcludeDisposition:
		return "Exclude"
	default:
		return "----"
	}
}

// Suffix returns the disposition marker as a suffix (appears after filename)
func (d FileDisposition) Suffix() string {
	return fmt.Sprintf("[%s]", d.Key())
}

// Key returns the disposition key
func (d FileDisposition) Key() (key string) {
	if d == 0 {
		return "?"
	}
	switch d {
	case CommitDisposition, OmitDisposition, GitIgnoreDisposition, GitExcludeDisposition:
		return string(d)
	}
	return "?"
}

// ParseFileDisposition parses a string to FileDisposition.
// Accepts (case-insensitive):
// - Single char: "c", "o", "i", "x"
// - String() values: "Commit", ".gitignore", ".git/info/exclude"
// - Label() values: "Commit", "Omit", "Ignore", "Exclude"
func ParseFileDisposition(s string) (fd FileDisposition, err error) {
	s = strings.TrimSpace(s)
	switch {
	case s == "":
		fd = UnspecifiedDisposition

	case len(s) == 1:
		switch s[0] {
		case 'c', 'C':
			fd = CommitDisposition
		case 'o', 'O':
			fd = OmitDisposition
		case 'i', 'I':
			fd = GitIgnoreDisposition
		case 'x', 'X':
			fd = GitExcludeDisposition
		default:
			err = NewErr(ErrInvalidFileDisposition, StringKV("value", s))
		}

	default:
		// Label() or String() matching
		switch strings.ToLower(s) {
		case "commit":
			fd = CommitDisposition
		case "omit":
			fd = OmitDisposition
		case "ignore", ".gitignore":
			fd = GitIgnoreDisposition
		case "exclude", ".git/info/exclude":
			fd = GitExcludeDisposition
		case "unspecified":
			fd = UnspecifiedDisposition
		default:
			err = NewErr(ErrInvalidFileDisposition, StringKV("value", s))
		}
	}
	return fd, err
}
