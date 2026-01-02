package gomtui

import (
	"fmt"
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

// IsFileDisposition returns true if the string passed matches a valid file disposition
func IsFileDisposition(fd FileDisposition) bool {
	return fd.Key() == string(fd)
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

// Color returns the lipgloss color for the disposition as a string value
func (d FileDisposition) Color() string {
	return string(d.RGBColor())
}

// RGBColor returns the lipgloss color for the disposition as an RGBColor value
func (d FileDisposition) RGBColor() RGBColor {
	switch d {
	case CommitDisposition:
		return GreenColor // Green
	case OmitDisposition:
		return GrayColor // Gray
	case GitIgnoreDisposition:
		return RedColor // Red
	case GitExcludeDisposition:
		return YellowColor // Yellow
	case UnspecifiedDisposition:
		fallthrough
	default:
		return WhiteColor // White
	}
}
