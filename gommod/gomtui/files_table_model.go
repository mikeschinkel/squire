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
// - isSelectedRow: Flag for row-level styling (bold + bright)
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
	dispositionFn func(dt.RelFilepath) FileDisposition
	currentColumn int // Current column index (0-based) for cell-level highlighting
}

// NewFilesTableModel creates a new files table model from a Directory.
// The Directory contains Files with metadata and a summary.
func NewFilesTableModel(dir Directory, dispositionFn func(dt.RelFilepath) FileDisposition, width, height int) FilesTableModel {
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

	// Create table with styling
	// We handle row highlighting via per-cell reverse-video styling (in buildTableRow)
	// Disable bubble-table's built-in row highlighting to avoid conflicts
	noHighlightStyle := lipgloss.NewStyle()

	t := table.New(columns).
		WithRows(rows).
		Focused(true).
		WithMaxTotalWidth(width).           // Don't exceed this width - enable scrolling when table is bigger
		WithTargetWidth(width).             // Fill available width when table is smaller
		WithPageSize(rowsNeeded).           // Set page size to fill height
		WithMinimumHeight(height).          // Force table to fill visual height
		WithFooterVisibility(false).        // Hide "1/1" pagination footer
		WithHorizontalFreezeColumnCount(1). // Freeze first column (#) when scrolling
		HeaderStyle(styleWithRGBColor(CyanColor).Bold(true)).
		SelectableRows(false).            // We don't need checkboxes, just highlighting
		HighlightStyle(noHighlightStyle). // Disable built-in highlighting - we handle it per-cell
		BorderRounded().
		WithBaseStyle(styleWithRGBColor(CyanColor))

	return FilesTableModel{
		table:         t,
		columns:       columns,
		dir:           dir,
		width:         width,
		height:        height,
		dispositionFn: dispositionFn,
		currentColumn: 0, // Start with first column selected
	}
}

// Init initializes the model.
func (m FilesTableModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model.
//
// Handles three types of interactions:
// 1. Cell navigation (left/right) - We handle AND conditionally transform for scrolling
// 2. Disposition keys (c/o/g/e) - We handle, emit changeDispositionMsg
// 3. Row navigation (up/down) - We delegate to bubble-table
//
// KEY MECHANISM: Message transformation for horizontal scrolling
// - When table fits in viewport: left/right move cell cursor only (we update currentColumn)
// - When cell cursor goes off-screen: Transform left→shift+left, right→shift+right
// - bubble-table receives shift+left/shift+right and triggers horizontal scroll
// - This gives us: cell cursor movement within visible columns + scrolling when needed
//
// Row rebuilding happens on EVERY selection change (row OR column OR disposition).
// This is O(n) but fast for typical directory sizes. See README.md for details.
func (m FilesTableModel) Update(msg tea.Msg) (FilesTableModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		keyMsg := msg.String()

		// CELL NAVIGATION: Handle left/right for cell-level navigation
		// We update currentColumn AND pass to bubble-table for horizontal scrolling
		switch keyMsg {
		case "left":
			// Move to previous column (no wrap-around at first column)
			if m.currentColumn > 0 {
				m.currentColumn--
				m = m.refreshTable()
			}
			// DON'T return early - fall through to pass to bubble-table for horizontal scroll

		case "right":
			// Move to next column (no wrap-around at last column)
			if m.currentColumn < numColumns-1 {
				m.currentColumn++
				m = m.refreshTable()
			}

			// DON'T return early - fall through to pass to bubble-table for horizontal scroll

		default:
			// DISPOSITION KEYS: Handle c/o/g/e for file disposition changes
			if len(keyMsg) == 1 {
				fd := FileDisposition(keyMsg[0])
				if fd.IsValid() {
					// Emit disposition change event (parent will update dispositions map)
					msg := m.maybeChangeDisposition(fd)
					// Rebuild table rows to reflect changes in disposition column colors
					// Preserve both row and column selection
					m = m.refreshTable()
					return m, func() tea.Msg { return msg }
				}
			}
		}
	}

	// DELEGATE TO BUBBLE-TABLE: Pass messages to bubble-table
	// For left/right: Transform to shift+left/shift+right when horizontal scrolling is needed
	// For up/down: Pass through unchanged for row navigation
	prevSelectedIdx := m.table.GetHighlightedRowIndex()

	// Check if we need horizontal scrolling (table wider than viewport)
	needsScroll := m.width > 0 && m.width < m.widthToColumn(m.currentColumn+1)

	// Transform left/right to shift+left/shift+right when scrolling is needed
	if keyMsg, ok := msg.(tea.KeyMsg); ok && needsScroll {
		switch keyMsg.String() {
		case "left":
			// Transform to shift+left (bubble-table's default scroll left key)
			msg = tea.KeyMsg{Type: tea.KeyShiftLeft}
		case "right":
			// Transform to shift+right (bubble-table's default scroll right key)
			msg = tea.KeyMsg{Type: tea.KeyShiftRight}
		}
		// Trigger recalculation
		m.table = m.table.WithMaxTotalWidth(m.width)
	}

	m.table, cmd = m.table.Update(msg)
	newSelectedIdx := m.table.GetHighlightedRowIndex()

	// If row selection changed, rebuild rows to update:
	// 1. ▶ indicator (moves to new row)
	// 2. Bold styling (applies to new row)
	// 3. Reverse-video (applies to current cell in new row)
	if prevSelectedIdx != newSelectedIdx {
		m.table = m.table.WithRows(buildTableRows(m.dir.Files, m.dispositionFn, newSelectedIdx, m.currentColumn))
	}

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
	selectedIdx := m.table.GetHighlightedRowIndex()
	rows := buildTableRows(m.dir.Files, m.dispositionFn, selectedIdx, m.currentColumn)

	// Calculate how many rows we need to fill the visual height
	// Table chrome includes: borders, column headers, footer (if visible)
	// See layout_constants.go for detailed breakdown
	rowsNeeded := height - TableChromeLines

	// Update table dimensions and rows
	m.table = m.table.
		WithTargetWidth(width).
		WithMaxTotalWidth(width).
		WithMinimumHeight(height).
		WithPageSize(rowsNeeded).
		WithRows(rows)
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

func (m FilesTableModel) widthToColumn(colNo int) int {
	width := m.width
	if colNo >= len(m.columns) {
		goto end
	}
	width = 2 + colNo + 1
	// Two for outer borders, colNo for dividers, and one just because.
	for i := 0; i < colNo; i++ {
		width += m.columns[i].Width()
	}
end:
	return width
}

func (m FilesTableModel) refreshTable() FilesTableModel {
	selectedIdx := m.table.GetHighlightedRowIndex()
	// Rebuild rows to update cell highlighting (reverse-video moves to new cell)
	rows := buildTableRows(m.dir.Files, m.dispositionFn, selectedIdx, m.currentColumn)
	m.table = m.table.WithRows(rows)
	return m
}

// maybeChangeDisposition returns a changeDispositionMsg for the currently selected file.
func (m FilesTableModel) maybeChangeDisposition(disp FileDisposition) (msg changeDispositionMsg) {
	cursor := m.table.GetHighlightedRowIndex()
	if cursor < 0 {
		goto end
	}

	if cursor >= len(m.dir.Files) {
		goto end
	}

	msg = maybeChangeDisposition(m.dir.Files[cursor].Path, disp)

end:
	return msg
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
func buildTableRows(files []*bubbletree.File, dispositionFn func(dt.RelFilepath) FileDisposition, selectedRowIndex, selectedColumnIndex int) []table.Row {
	rows := make([]table.Row, 0, len(files))

	for i, file := range files {
		row := buildTableRow(i+1, file, dispositionFn, i, selectedRowIndex, selectedColumnIndex)
		rows = append(rows, row)
	}

	return rows
}

// buildTableRow creates a single table row from a File with per-cell styling.
// rowIndex is 0-based row index, selectedRowIndex indicates selected row, selectedColumnIndex indicates selected column.
func buildTableRow(rowNum int, file *bubbletree.File, dispositionFn func(dt.RelFilepath) FileDisposition, rowIndex, selectedRowIndex, selectedColumnIndex int) table.Row {
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
func styledDispositionCell(disp FileDisposition, isSelectedRow bool, isCurrentCell bool) table.StyledCell {
	// Get disposition's semantic color and adjust for selection
	color := adjustColorForSelection(disp.RGBColor(), isSelectedRow)

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
	if label == GitExcludeDisposition.Label() {
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
