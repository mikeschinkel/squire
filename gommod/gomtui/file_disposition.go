package gomtui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
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

func maybeGetFileDisposition(msg tea.Msg) (fd FileDisposition) {
	var result string
	// Handle disposition change keys (c/o/g/e)
	keyMsg, err := dtx.AssertType[tea.KeyMsg](msg)
	if err != nil {
		goto end
	}
	result = keyMsg.String()
	if len(result) != 1 {
		goto end
	}
	fd = FileDisposition(result[0])
end:
	return fd
}

func maybeChangeDispositionMsg(rfp dt.RelFilepath, fd FileDisposition) (cdm changeDispositionMsg) {
	if !fd.IsValid() {
		goto end
	}
	cdm = changeDispositionMsg{
		Path:        rfp,  // Table view only handles files, not directories
		Disposition: fd,
	}
end:
	return cdm
}
