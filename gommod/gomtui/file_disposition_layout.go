package gomtui

// FileDispositionLayout manages dimension calculations for the file disposition view.
// This view has: header, tree pane (left), content/table pane (right), footer.
type FileDispositionLayout struct {
	terminalWidth  int
	terminalHeight int
	leftPaneWidth  int // Actual tree content width
}

// NewFileDispositionLayout creates a layout manager for the given terminal dimensions.
func NewFileDispositionLayout(terminalWidth, terminalHeight, leftPaneWidth int) FileDispositionLayout {
	return FileDispositionLayout{
		terminalWidth:  terminalWidth,
		terminalHeight: terminalHeight,
		leftPaneWidth:  leftPaneWidth,
	}
}

// LeftPaneWidth returns the width for the left tree pane.
func (l FileDispositionLayout) LeftPaneWidth() int {
	return l.leftPaneWidth
}

// RightPaneWidth returns the total width for the right pane including chrome.
// Used when rendering file content (which needs explicit width for pane wrapper).
func (l FileDispositionLayout) RightPaneWidth() int {
	return l.terminalWidth - l.leftPaneWidth
}

// RightPaneInnerWidth returns the width for the right content/table pane.
// Calculation: terminal width - tree width - borders(2) - padding(2) - spacing(6)
// The -10 is empirically determined (not theoretically derivable from lipgloss).
func (l FileDispositionLayout) RightPaneInnerWidth() int {
	return l.RightPaneWidth() - 10
}

// PaneHeight returns the full pane height (outer, including borders).
// Used by: table (direct render), basePaneStyle wrapper
// Calculation: terminal height - header(1) - footer(1) - spacing(2)
func (l FileDispositionLayout) PaneHeight() int {
	return l.terminalHeight - 2
}

// PaneInnerHeight returns the viewport height inside a pane (inner, excluding pane borders).
// Used by: tree viewport, file content viewport
// Calculation: pane layout height - top border(1) - bottom border(1)
func (l FileDispositionLayout) PaneInnerHeight() int {
	return l.PaneHeight() - 2
}
