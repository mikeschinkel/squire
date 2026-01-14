package gomtui

// FilesTableModel provides a rich, interactive table view for directory file listings.
//
// This table implements a two-level selection system (row + cell) with horizontal scrolling,
// frozen columns, and custom styling. See gomtui/README.md for comprehensive documentation on:
//
// - Two-level selection system (row highlighting + cell cursor)
// - Width calculation challenges (lipgloss version mismatch)
// - Styling strategy (HSL color adjustment, reverse-video)
// - Navigation implementation (why we intercept left/right)
// - Performance considerations (row rebuilding on every change)
//
// Quick reference:
// - up/down: Row navigation (bubble-table)
// - left/right: Cell navigation (we handle, not bubble-table)
// - c/o/g/e: Set file disposition
// - currentColumn: Tracks cell-level cursor position
// - isSelectedRow: Flag for row-level styling (bold + bright)GetAllDescendantPaths
// - isCurrentCell: Flag for cell-level styling (reverse-video)

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/gomion/gommod/bubbletree"
	"github.com/mikeschinkel/gomion/gommod/gompkg"
)

// Column keys for bubble-table
const (
	colKeyRowNum   = "num"
	colKeyFilename = "filename"
	colKeyDisp     = "disp"
	colKeyChange   = "change"
	colKeySize     = "size"
	colKeyModified = "modified"
	colKeyPerms    = "perms"
	colKeyMode     = "mode"
	colKeyFlags    = "flags"
)

// Column indices (for cell-level highlighting)
const (
	colIndexRowNum = iota
	colIndexDisp
	colIndexFilename
	colIndexChange
	colIndexSize
	colIndexModified
	colIndexPerms
	colIndexMode
	colIndexFlags
	numColumns // Total number of columns
)

// FilesTableModel wraps bubble-table for displaying directory file lists with metadata.
type FilesTableModel struct {
	table         table.Model
	columns       []table.Column
	dir           Directory // Directory being displayed
	width         int
	height        int
	dispositionFn func(dt.RelFilepath) gompkg.FileDisposition
	// Note: Cell cursor column index is now tracked by bubble-table internally
	// Use m.table.GetCellCursorColumnIndex() to retrieve it
}

// NewFilesTableModel creates a new files table model from a Directory.
// The Directory contains Files with metadata and a summary.
func NewFilesTableModel(dir Directory, dispositionFn func(dt.RelFilepath) gompkg.FileDisposition, width, height int) FilesTableModel {
	// Calculate dynamic filename column width
	filenameWidth := calculateMaxFilenameWidth(dir.Files)

	// Define table columns
	center := lipglossStyle.Align(lipgloss.Center)
	leftAligned := lipglossStyle.Align(lipgloss.Left).PaddingLeft(1)

	columns := []table.Column{
		table.NewColumn(colKeyRowNum, "#", 5).WithStyle(center), // Includes ▶ indicator
		table.NewColumn(colKeyDisp, "Plan", 8).WithStyle(center),
		table.NewColumn(colKeyFilename, " Filename", filenameWidth), // Fixed width based on max filename
		table.NewColumn(colKeyChange, "Change", 11).WithStyle(center),
		table.NewColumn(colKeySize, "Size   ", 12).WithStyle(lipglossStyle.Align(lipgloss.Right)),
		table.NewColumn(colKeyModified, "Modified", 21).WithStyle(center),
		table.NewColumn(colKeyPerms, "Perms", 11).WithStyle(center),
		table.NewColumn(colKeyMode, "Mode", 6).WithStyle(center),
		table.NewFlexColumn(colKeyFlags, "Flags", 1).WithStyle(leftAligned), // Flex column, left-aligned with padding
	}

	// Build table rows from directory files
	// Initially, first row (index 0) and first column (index 0) are selected
	rows := buildTableRows(dir.Files, dispositionFn, 0, 0)

	// We only need as many rows as we have filesYou
	rowsNeeded := len(dir.Files)

	// Calculate natural width of table (sum of column widths + borders)
	// This is what bubble-table does in recalculateWidth() when targetTotalWidth=0
	naturalWidth := 0
	for _, col := range columns {
		naturalWidth += col.Width()
	}
	naturalWidth += len(columns) + 1 // Add borders (1 at start + 1 between each + 1 at end)

	// Create table with styling
	// We handle row highlighting via per-cell reverse-video styling (in buildTableRow)
	// Disable bubble-table's built-in row highlighting to avoid conflicts
	noHighlightStyle := lipgloss.NewStyle()

	t := table.New(columns).
		WithRows(rows).
		Focused(true).
		WithCellCursorMode(true).           // Enable cell cursor mode (left/right navigate cells, not pages)
		WithMaxTotalWidth(width).           // Viewport width constraint - enables scrolling when columns exceed this
		WithPageSize(rowsNeeded).           // Set page size to fill height
		WithMinimumHeight(height).          // Force table to fill visual height
		WithFooterVisibility(false).        // Hide "1/1" pagination footer
		WithHorizontalFreezeColumnCount(1). // Freeze first column (#) when scrolling
		HeaderStyle(styleWithRGBColor(CyanColor).Bold(true)).
		SelectableRows(false).            // We don't need checkboxes, just highlighting
		HighlightStyle(noHighlightStyle). // Disable built-in highlighting - we handle it per-cell
		BorderRounded().
		WithBaseStyle(styleWithRGBColor(CyanColor))

	// Only use WithTargetWidth when table fits in viewport (to fill space)
	// When table exceeds viewport, omit WithTargetWidth to enable horizontal scrolling
	if naturalWidth <= width {
		t = t.WithTargetWidth(width)
	}

	return FilesTableModel{
		table:         t,
		columns:       columns,
		dir:           dir,
		width:         width,
		height:        height,
		dispositionFn: dispositionFn,
	}
}

// Init initializes the model.
func (m FilesTableModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model.
//
// Handles two types of interactions:
// 1. Disposition keys (c/o/g/e) - We handle, emit requestDispositionChangeCmd
// 2. All other keys - Delegated to bubble-table (navigation, scrolling, etc.)
//
// Row rebuilding happens on EVERY change to update cell highlighting.
// This is O(n) but fast for typical directory sizes. See README.md for details.
func (m FilesTableModel) Update(msg tea.Msg) (FilesTableModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle disposition change keys (c/o/g/e)
		if fd := extractDispositionFromKeyMsg(msg); fd.IsValid() {
			// Emit disposition change event (parent will update dispositions map and batch with refreshTableCmd)
			return m, m.requestDispositionChangeCmd(fd)

		}
		// Fall through to delegate to bubble-table

	case refreshTableMsg:
		// Rebuild table rows (triggered after disposition changes)
		m = m.refreshTable()
		return m, nil
	}

	// Delegate all other messages to bubble-table (navigation, scrolling, etc.)
	m.table, cmd = m.table.Update(msg)

	// Always rebuild rows to update cell cursor highlighting
	// Get current selection state from bubble-table
	selectedRowIdx := m.table.GetHighlightedRowIndex()
	selectedColIdx := m.table.GetCellCursorColumnIndex()

	m.table = m.table.WithRows(buildTableRows(m.dir.Files, m.dispositionFn, selectedRowIdx, selectedColIdx))

	return m, cmd
}

// View renders the table directly (no summary - table columns show all the info).
func (m FilesTableModel) View() string {
	return m.table.View()
}

// SetSize updates the model dimensions.
func (m FilesTableModel) SetSize(width, height int) FilesTableModel {
	m.width = width
	m.height = height

	// Rebuild rows with padding for new height, preserving selection
	selectedRowIdx := m.table.GetHighlightedRowIndex()
	selectedColIdx := m.table.GetCellCursorColumnIndex()
	rows := buildTableRows(m.dir.Files, m.dispositionFn, selectedRowIdx, selectedColIdx)

	// Calculate how many rows we need to fill the visual height
	// Table chrome includes: borders, column headers, footer (if visible)
	// See layout_constants.go for detailed breakdown
	rowsNeeded := height - TableChromeLines

	// Calculate natural width of table
	naturalWidth := 0
	for _, col := range m.columns {
		naturalWidth += col.Width()
	}
	naturalWidth += len(m.columns) + 1

	// Update table dimensions and rows
	m.table = m.table.
		WithMaxTotalWidth(width).
		WithMinimumHeight(height).
		WithPageSize(rowsNeeded).
		WithRows(rows)

	// Only use WithTargetWidth when table fits in viewport
	if naturalWidth <= width {
		m.table = m.table.WithTargetWidth(width)
	}

	return m
}

// GetSelectedFile returns the currently selected file.
func (m FilesTableModel) GetSelectedFile() *bubbletree.File {
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

func (m FilesTableModel) refreshTable() FilesTableModel {
	selectedRowIdx := m.table.GetHighlightedRowIndex()
	selectedColIdx := m.table.GetCellCursorColumnIndex()
	// Rebuild rows to update cell highlighting (reverse-video moves to new cell)
	rows := buildTableRows(m.dir.Files, m.dispositionFn, selectedRowIdx, selectedColIdx)
	m.table = m.table.WithRows(rows)
	return m
}

// requestDispositionChangeCmd returns a changeDispositionMsg for the currently selected file.
func (m FilesTableModel) requestDispositionChangeCmd(fd gompkg.FileDisposition) (cmd tea.Cmd) {

	cursor := m.table.GetHighlightedRowIndex()
	if cursor < 0 {
		goto end
	}

	if cursor >= len(m.dir.Files) {
		goto end
	}

	if !fd.IsValid() {
		goto end
	}

	cmd = requestDispositionChangeCmd(m.dir.Files[cursor].Path, fd) // Table view only handles files, not directories

end:
	return cmd
}

// calculateMaxFilenameWidth returns the width needed for the filename column
// based on the longest filename in the file list, with minimum padding.
func calculateMaxFilenameWidth(files []*bubbletree.File) int {
	const minWidth = 8 // Minimum width for "Filename" header
	const padding = 2  // 1 char left + 1 char right

	maxLen := minWidth
	for _, file := range files {
		nameLen := len(file.Path.Base())
		if nameLen > maxLen {
			maxLen = nameLen
		}
	}

	return maxLen + padding
}

// buildTableRows converts Files with metadata into table rows with styled cells.
// selectedRowIndex is the 0-based index of the selected row (-1 for none).
// selectedColumnIndex is the 0-based index of the selected column.
func buildTableRows(files []*bubbletree.File, dispositionFn func(dt.RelFilepath) gompkg.FileDisposition, selectedRowIndex, selectedColumnIndex int) []table.Row {
	rows := make([]table.Row, 0, len(files))

	for i, file := range files {
		row := buildTableRow(i+1, file, dispositionFn, i, selectedRowIndex, selectedColumnIndex)
		rows = append(rows, row)
	}

	return rows
}

// buildTableRow creates a single table row from a File with per-cell styling.
// rowIndex is 0-based row index, selectedRowIndex indicates selected row, selectedColumnIndex indicates selected column.
func buildTableRow(rowNum int, file *bubbletree.File, dispositionFn func(dt.RelFilepath) gompkg.FileDisposition, rowIndex, selectedRowIndex, selectedColumnIndex int) table.Row {
	var size, modified, perms, mode, flags string

	// Default values for missing metadata
	size = "N/A"
	modified = "N/A"
	perms = "N/A"
	mode = "N/A"
	flags = ""

	// Populate from metadata if available
	if file.HasMeta() {
		meta := file.Meta()
		// Size - human readable
		size = formatFileSize(meta.Size)

		// Modified time - YYYY-MM-DD HH:MM:SS
		modified = meta.ModTime.Format("2006-01-02 15:04:05")

		// Permissions - symbolic (rwxr-xr-x)
		perms = formatPermissions(meta.Permissions)

		// Mode - numeric (755, 644, etc.)
		mode = fmt.Sprintf("%04o", meta.Permissions&0777)

		// Flags - x for executable, l for symlink
		flags = formatFlags(meta.EntryStatus, meta.Permissions)
	}

	disposition := dispositionFn(file.Path)

	// Check if this is the selected row (for ▶ indicator)
	isSelectedRow := rowIndex == selectedRowIndex

	// Combine indicator and row number in a single cell
	// Selected single digit: "▶ 1" (3 chars)
	// Selected double digit: "▶10" (3 chars)
	// Unselected single digit: "  1" (3 chars - two spaces)
	// Unselected double digit: " 10" (3 chars - one space)
	var rowNumStr string
	if isSelectedRow {
		if rowNum < 10 {
			rowNumStr = fmt.Sprintf("▶ %d", rowNum)
		} else {
			rowNumStr = fmt.Sprintf("▶%d", rowNum)
		}
	} else {
		if rowNum < 10 {
			rowNumStr = fmt.Sprintf("  %d", rowNum)
		} else {
			rowNumStr = fmt.Sprintf(" %d", rowNum)
		}
	}

	// Create styled cells
	// - Entire selected row gets: bold + brighter colors (isSelectedRow)
	// - Current cell (selected row AND current column) also gets: reverse-video
	rowNumCell := styledMetadataCell(rowNumStr, WhiteColor, isSelectedRow, isSelectedRow && selectedColumnIndex == colIndexRowNum)
	dispCell := styledDispositionCell(disposition, isSelectedRow, isSelectedRow && selectedColumnIndex == colIndexDisp)
	changeCell := styledChangeCell(file, isSelectedRow, isSelectedRow && selectedColumnIndex == colIndexChange)

	// Create styled cells for metadata
	// Add left padding to filename
	filenameCell := styledMetadataCell(string(file.Path.Base()), WhiteColor, isSelectedRow, isSelectedRow && selectedColumnIndex == colIndexFilename)
	filenameCell.Style = filenameCell.Style.PaddingLeft(1)
	sizeCell := styledMetadataCell(size, WhiteColor, isSelectedRow, isSelectedRow && selectedColumnIndex == colIndexSize)
	modifiedCell := styledMetadataCell(modified, WhiteColor, isSelectedRow, isSelectedRow && selectedColumnIndex == colIndexModified)
	permsCell := styledMetadataCell(perms, WhiteColor, isSelectedRow, isSelectedRow && selectedColumnIndex == colIndexPerms)
	modeCell := styledMetadataCell(mode, WhiteColor, isSelectedRow, isSelectedRow && selectedColumnIndex == colIndexMode)
	flagsCell := styledMetadataCell(flags, WhiteColor, isSelectedRow, isSelectedRow && selectedColumnIndex == colIndexFlags)

	return table.NewRow(table.RowData{
		colKeyRowNum:   rowNumCell,
		colKeyDisp:     dispCell,
		colKeyFilename: filenameCell,
		colKeyChange:   changeCell,
		colKeySize:     sizeCell,
		colKeyModified: modifiedCell,
		colKeyPerms:    permsCell,
		colKeyMode:     modeCell,
		colKeyFlags:    flagsCell,
	})
}

// adjustColorForSelection returns a brighter or dimmed version of a color based on selection state.
//
// Uses HSL color space to adjust brightness while preserving hue and saturation.
// This keeps semantic colors recognizable (green=COMMIT, yellow=OMIT, etc.) while
// providing clear visual distinction between selected and unselected rows.
//
// - Selected rows: Lightness +40% (capped at 0.8)
// - Unselected rows: Lightness reduced to 60% (floored at 0.2)
//
// Example: Green #00FF00 → Selected: #66FF66, Unselected: #009900
// See README.md "Styling Strategy" for detailed explanation.
func adjustColorForSelection(rgbColor RGBColor, isSelected bool) string {
	// Parse the hex color string (e.g., "#00FF00")
	c, err := colorful.Hex(string(rgbColor))
	if err != nil {
		// If parsing fails, return original color unchanged
		return string(rgbColor)
	}

	// Convert to HSL to preserve hue and saturation, only adjust lightness
	// HSL: Hue (color), Saturation (intensity), Lightness (brightness)
	h, s, l := c.Hsl()

	if isSelected {
		// Make selected rows brighter (increase lightness)
		// Formula: l = l + (1.0 - l) * 0.4
		// This brightens by 40% of remaining headroom to white
		// Example: l=0.5 → l=0.7 (50% + 40% of 50% remaining)
		l = l + (1.0-l)*0.4
		// Cap at 0.8 to avoid washing out to near-white
		if l > 0.8 {
			l = 0.8
		}
	} else {
		// Make non-selected rows dimmer (decrease lightness)
		// Reduce lightness to 60% of original value
		// Example: l=0.5 → l=0.3 (60% of original brightness)
		l = l * 0.6
		// Floor at 0.2 to maintain readability (not too dark)
		if l < 0.2 {
			l = 0.2
		}
	}

	// Convert back to RGB and return as hex string
	adjusted := colorful.Hsl(h, s, l)
	return adjusted.Hex()
}

// styledMetadataCell returns a styled cell for metadata columns (filename, size, permissions, etc.).
//
// Implements two-level highlighting:
// - isSelectedRow: Bold + brighter color (entire row)
// - isCurrentCell: Also reverse-video (one cell in selected row)
//
// Only ONE cell in the entire table has reverse-video at a time (the "cursor").
// See README.md "Two-Level Selection System" for details.
func styledMetadataCell(text string, baseColor RGBColor, isSelectedRow bool, isCurrentCell bool) table.StyledCell {
	// Adjust color brightness based on selection
	color := adjustColorForSelection(baseColor, isSelectedRow)

	// Create style: colored text with explicit dark background, bold if selected row
	// Background must be set explicitly for Reverse to work visibly
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Background(lipgloss.Color("#1a1a1a")).
		Bold(isSelectedRow)

	// Add reverse-video if this is the current cell (cell-level cursor)
	// Reverse swaps foreground and background colors
	if isCurrentCell {
		style = style.Reverse(true)
	}

	return table.NewStyledCell(text, style)
}

// styledDispositionCell returns a styled cell for disposition column with semantic colors.
//
// Disposition colors: COMMIT=green, OMIT=yellow, IGNORE/EXCLUDE=red
// Applies same two-level highlighting as styledMetadataCell, using disposition's
// semantic color as the base.
//
// Special case: EXCLUDE label gets leading space for alignment.
// See README.md "Two-Level Selection System" for details.
func styledDispositionCell(disp gompkg.FileDisposition, isSelectedRow bool, isCurrentCell bool) table.StyledCell {
	// Get disposition's semantic color and adjust for selection
	color := adjustColorForSelection(DispositionColor(disp), isSelectedRow)

	// Create style: colored text with explicit dark background, bold if selected row
	// Background must be set explicitly for Reverse to work visibly
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Background(lipgloss.Color("#1a1a1a")).
		Bold(isSelectedRow)

	// Add reverse-video if this is the current cell
	// Reverse swaps foreground and background colors
	if isCurrentCell {
		style = style.Reverse(true)
	}

	// Get disposition label and add padding if needed
	label := disp.Label()
	if label == gompkg.GitExcludeDisposition.Label() {
		label = " " + label // Add leading space for alignment
	}

	return table.NewStyledCell(label, style)
}

// styledStatusCell returns a styled cell for git staging status with color.
func styledStatusCell(metadata *FileMeta) table.StyledCell {
	if metadata == nil {
		return table.NewStyledCell("---", lipgloss.NewStyle())
	}
	color := StagingRGBColor(metadata.Staging)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	return table.NewStyledCell(metadata.Staging.Label(), style)
}

// styledChangeCell returns a styled cell for change type with color.
// isSelectedRow: entire row gets bold + brighter colors
// isCurrentCell: this specific cell also gets reverse-video
func styledChangeCell(file *bubbletree.File, isSelectedRow bool, isCurrentCell bool) table.StyledCell {
	//meta := file.Meta()

	if !file.HasData() {
		return table.NewStyledCell("---", lipgloss.NewStyle())
	}

	fileData, err := dtx.AssertType[*FileData](file.Data())
	if err != nil {
		panic(err.Error())
	}

	status := fileData.FileStatus

	// Use unstaged change if available, otherwise staged
	changeType := status.UnstagedChange
	if changeType == 0 {
		changeType = status.StagedChange
	}

	color := adjustColorForSelection(ChangeTypeRGBColor(changeType), isSelectedRow)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(isSelectedRow)

	if isCurrentCell {
		style = style.Reverse(true)
	}

	return table.NewStyledCell(changeType.String(), style)
}

// formatFileSize converts bytes to human-readable format.
func formatFileSize(bytes int64) (size string) {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B   ", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB ", float64(bytes)/float64(div), "KMGTPE"[exp])
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
