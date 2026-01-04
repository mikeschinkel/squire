package gomtui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/mikeschinkel/go-dt"
)

// Column keys for bubble-table
const (
	colKeyRowNum   = "num"
	colKeyFilename = "filename"
	colKeyDisp     = "disp"
	colKeyStatus   = "status"
	colKeyChange   = "change"
	colKeySize     = "size"
	colKeyModified = "modified"
	colKeyPerms    = "perms"
	colKeyMode     = "mode"
	colKeyFlags    = "flags"
)

// FilesTableModel wraps bubble-table for displaying directory file lists with metadata.
type FilesTableModel struct {
	table  table.Model
	dir    *Directory // Directory being displayed
	width  int
	height int
}

// NewFilesTableModel creates a new files table model from a Directory.
// The Directory contains Files with metadata and a summary.
func NewFilesTableModel(dir *Directory, width, height int) FilesTableModel {
	// Define table columns
	columns := []table.Column{
		table.NewColumn(colKeyRowNum, "#", 4),
		table.NewColumn(colKeyFilename, "Filename", 30),
		table.NewColumn(colKeyDisp, "Disp", 6),
		table.NewColumn(colKeyStatus, "Status", 10),
		table.NewColumn(colKeyChange, "Change", 8),
		table.NewColumn(colKeySize, "Size", 10),
		table.NewColumn(colKeyModified, "Modified", 20),
		table.NewColumn(colKeyPerms, "Perms", 12),
		table.NewColumn(colKeyMode, "Mode", 6),
		table.NewColumn(colKeyFlags, "Flags", 6),
	}

	// Build table rows from directory files
	rows := buildTableRows(dir.Files)

	// We only need as many rows as we have files
	rowsNeeded := len(dir.Files)

	// Create table with styling
	t := table.New(columns).
		WithRows(rows).
		Focused(true).
		WithTargetWidth(width).      // Set table width - CRITICAL for rendering!
		WithPageSize(rowsNeeded).    // Set page size to fill height
		WithMinimumHeight(height).   // Force table to fill visual height
		WithFooterVisibility(false). // Hide "1/1" pagination footer
		HeaderStyle(styleWithRGBColor(CyanColor).Bold(true)).
		SelectableRows(false). // We don't need checkboxes, just highlighting
		BorderRounded().
		WithBaseStyle(styleWithRGBColor(CyanColor))

	return FilesTableModel{
		table:  t,
		dir:    dir,
		width:  width,
		height: height,
	}
}

// buildTableRows converts Files with metadata into table rows with styled cells.
func buildTableRows(files []*File) []table.Row {
	rows := make([]table.Row, 0, len(files))

	for i, file := range files {
		row := buildTableRow(i+1, file)
		rows = append(rows, row)
	}

	return rows
}

// buildTableRow creates a single table row from a File with per-cell styling.
func buildTableRow(rowNum int, file *File) table.Row {
	var size, modified, perms, mode, flags string

	// Default values for missing metadata
	size = "N/A"
	modified = "N/A"
	perms = "N/A"
	mode = "N/A"
	flags = ""

	// Populate from metadata if available
	if file.Metadata != nil {
		// Size - human readable
		size = formatFileSize(file.Metadata.Size)

		// Modified time - YYYY-MM-DD HH:MM:SS
		modified = file.Metadata.ModTime.Format("2006-01-02 15:04:05")

		// Permissions - symbolic (rwxr-xr-x)
		perms = formatPermissions(file.Metadata.Permissions)

		// Mode - numeric (755, 644, etc.)
		mode = fmt.Sprintf("%04o", file.Metadata.Permissions&0777)

		// Flags - x for executable, l for symlink
		flags = formatFlags(file.Metadata.EntryStatus, file.Metadata.Permissions)
	}

	// DEBUG: Use plain strings first to test if StyledCell is the issue
	dispStr := file.Disposition.Key()
	statusStr := "---"
	changeStr := "---"
	if file.Metadata != nil {
		statusStr = file.Metadata.Staging.Label()
		changeType := file.Metadata.UnstagedChange
		if changeType == 0 {
			changeType = file.Metadata.StagedChange
		}
		changeStr = changeType.Label()
	}

	return table.NewRow(table.RowData{
		colKeyRowNum:   fmt.Sprintf("%d", rowNum),
		colKeyFilename: string(file.Path),
		colKeyDisp:     dispStr,
		colKeyStatus:   statusStr,
		colKeyChange:   changeStr,
		colKeySize:     size,
		colKeyModified: modified,
		colKeyPerms:    perms,
		colKeyMode:     mode,
		colKeyFlags:    flags,
	})
}

// styledDispositionCell returns a styled cell for disposition with color.
func styledDispositionCell(disp FileDisposition) table.StyledCell {
	color := disp.RGBColor()
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	return table.NewStyledCell(disp.Key(), style)
}

// styledStatusCell returns a styled cell for git staging status with color.
func styledStatusCell(metadata *FileMetadata) table.StyledCell {
	if metadata == nil {
		return table.NewStyledCell("---", lipgloss.NewStyle())
	}
	color := StagingRGBColor(metadata.Staging)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	return table.NewStyledCell(metadata.Staging.Label(), style)
}

// styledChangeCell returns a styled cell for change type with color.
func styledChangeCell(metadata *FileMetadata) table.StyledCell {
	if metadata == nil {
		return table.NewStyledCell("---", lipgloss.NewStyle())
	}

	// Use unstaged change if available, otherwise staged
	changeType := metadata.UnstagedChange
	if changeType == 0 {
		changeType = metadata.StagedChange
	}

	color := ChangeTypeRGBColor(changeType)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	return table.NewStyledCell(changeType.Label(), style)
}

// formatFileSize converts bytes to human-readable format.
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatPermissions converts os.FileMode to symbolic notation (rwxr-xr-x).
func formatPermissions(mode os.FileMode) string {
	var b [9]byte
	w := 0
	for i := 0; i < 3; i++ {
		shift := uint(6 - i*3)
		if mode&(1<<(shift+2)) != 0 {
			b[w] = 'r'
		} else {
			b[w] = '-'
		}
		w++
		if mode&(1<<(shift+1)) != 0 {
			b[w] = 'w'
		} else {
			b[w] = '-'
		}
		w++
		if mode&(1<<shift) != 0 {
			b[w] = 'x'
		} else {
			b[w] = '-'
		}
		w++
	}
	return string(b[:])
}

// formatFlags returns symbolic flags (x for executable, l for symlink).
func formatFlags(status dt.EntryStatus, mode os.FileMode) (flags string) {
	if status == dt.IsSymlinkEntry {
		flags += "l"
	}
	if mode&0111 != 0 {
		flags += "x"
	}

	return flags
}

// Init initializes the model.
func (m FilesTableModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model.
// Handles disposition keys (c, o, g, e) and delegates navigation to table.
func (m FilesTableModel) Update(msg tea.Msg) (FilesTableModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Handle disposition keys
		if len(key) == 1 {
			fd := FileDisposition(key[0])
			if IsFileDisposition(fd) {
				// Apply disposition to selected file
				m.setDisposition(fd)
				// Rebuild table rows to reflect changes
				m.table = m.table.WithRows(buildTableRows(m.dir.Files))
				return m, nil
			}
		}
	}

	// Delegate to table for navigation
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// setDisposition applies a disposition to the currently selected file.
func (m *FilesTableModel) setDisposition(disp FileDisposition) {
	cursor := m.table.GetHighlightedRowIndex()
	if cursor >= 0 && cursor < len(m.dir.Files) {
		m.dir.Files[cursor].Disposition = disp
		// Recalculate summary
		if m.dir.Summary != nil {
			*m.dir.Summary = calculateDirSummary(m.dir.Files)
		}
	}
}

// View renders the table directly (no summary - table columns show all the info).
func (m FilesTableModel) View() string {
	return m.table.View()
}

// renderSummary renders the summary header with 4 summary statistics using renderRGBColor.
func (m FilesTableModel) renderSummary() string {
	s := m.dir.Summary

	// Format: "Files: 10 | Size: 1.2 MiB | Staged: 5 | Unstaged: 3"
	summaryText := fmt.Sprintf(
		"Files: %d | Size: %s | Disposition: C:%d O:%d G:%d E:%d | Status: Staged:%d Unstaged:%d Untracked:%d | Changes: M:%d A:%d D:%d R:%d",
		s.TotalFiles,
		formatFileSize(s.TotalSize),
		s.CommitCount,
		s.OmitCount,
		s.GitIgnoreCount,
		s.GitExcludeCount,
		s.StagedCount,
		s.UnstagedCount,
		s.UntrackedCount,
		s.ModifiedCount,
		s.AddedCount,
		s.DeletedCount,
		s.RenamedCount,
	)

	return renderRGBColor(summaryText, SilverColor)
}

// SetSize updates the model dimensions.
func (m FilesTableModel) SetSize(width, height int) FilesTableModel {
	m.width = width
	m.height = height

	// Rebuild rows with padding for new height
	rows := buildTableRows(m.dir.Files)

	// Calculate how many rows we need to fill the visual height
	// Table has: top border (1) + header (1) + bottom border (1) = 3 lines chrome
	rowsNeeded := len(m.dir.Files)

	// Update table dimensions and rows
	m.table = m.table.
		WithTargetWidth(width).
		WithMinimumHeight(height).
		WithPageSize(rowsNeeded).
		WithRows(rows)
	return m
}

// GetSelectedFile returns the currently selected file.
func (m FilesTableModel) GetSelectedFile() *File {
	cursor := m.table.GetHighlightedRowIndex()
	if cursor >= 0 && cursor < len(m.dir.Files) {
		return m.dir.Files[cursor]
	}
	return nil
}

// SetBorderColor updates the table border color (for focus indication).
func (m FilesTableModel) SetBorderColor(color RGBColor) FilesTableModel {
	m.table = m.table.WithBaseStyle(
		lipgloss.NewStyle().BorderForeground(lipgloss.Color(color)),
	)
	return m
}
