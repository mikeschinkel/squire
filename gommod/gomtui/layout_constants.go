package gomtui

// Layout dimension constants for UI chrome calculations.
//
// These constants define the fixed overhead for various UI components,
// making height calculations explicit and maintainable.
//
// View structure (from top to bottom):
//   Header (ViewHeaderLines)
//   \n
//   Body pane (variable height)
//   \n
//   Footer (ViewFooterLines)
//
// Pane structure:
//   Top border (1 line)
//   Content (variable height)
//   Bottom border (1 line)
//
// Table structure (rendered within a pane):
//   Top border (1 line)
//   Column headers (1 line)
//   Data rows (variable)
//   Bottom border (1 line)
//   [Footer - hidden in our case]

const (
	// View-level chrome (FileDispositionModel.View())
	ViewHeaderLines  = 1 // "Commit Plan: ..." header
	ViewFooterLines  = 1 // Menu footer with key bindings
	ViewSpacingLines = 2 // Newlines between header, body, footer

	// Pane chrome (lipgloss borders)
	PaneBorderLines = 2 // Top border + bottom border

	// Table chrome (bubble-table borders and headers)
	TableBorderLines = 2 // Top border + bottom border
	TableHeaderLines = 1 // Column headers row
	TableFooterLines = 1 // Pagination footer (we hide this with WithFooterVisibility(false))
)

// Derived constants for common calculations

const (
	// ViewChromeLines is the total fixed overhead for the entire view.
	// Used for: calculating available space for panes
	// Calculation: header + footer + spacing between sections
	ViewChromeLines = ViewHeaderLines + ViewFooterLines + ViewSpacingLines // = 4

	// TableChromeLines is the total fixed overhead for table rendering.
	// Used for: calculating how many data rows fit in table height
	// Calculation: borders + headers (footer is hidden)
	TableChromeLines = TableBorderLines + TableHeaderLines + TableFooterLines // = 4

	// PaneHeightForDirectoryView is the overhead to subtract from terminal height
	// when calculating pane height for directory view (table).
	// Used in: PaneHeight() when IsDirectoryView == true
	// Calculation: header + footer (no spacing needed for table)
	PaneHeightForDirectoryView = ViewHeaderLines + ViewFooterLines // = 2

	// PaneHeightForFileView is the overhead to subtract from terminal height
	// when calculating pane height for file content view.
	// Used in: PaneHeight() when IsDirectoryView == false
	// Calculation: header + footer + spacing
	PaneHeightForFileView = ViewChromeLines // = 4
)
