# gomtui - Gomion Terminal User Interface

This package provides the terminal user interface (TUI) for Gomion, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Table of Contents

- [FilesTableModel Overview](#filestablemodel-overview)
- [Two-Level Selection System](#two-level-selection-system)
- [Width Calculation Challenges](#width-calculation-challenges)
- [Styling Strategy](#styling-strategy)
- [Navigation Implementation](#navigation-implementation)
- [Performance Considerations](#performance-considerations)

---

## FilesTableModel Overview

`FilesTableModel` provides a rich, interactive table view for directory file listings with metadata (size, permissions, modification time, etc.).

### Key Features

#### Two-Level Selection (Row + Cell)
- **Row selection**: bubble-table tracks the selected row (up/down arrows)
- **Cell selection**: We track `currentColumn` for cell-level highlighting (left/right arrows)
- Selected row gets: **bold text + brighter colors** (entire row)
- Current cell gets: **reverse-video** (one cell within selected row)

#### Horizontal Scrolling with Frozen Column
- Table can be wider than viewport (all columns have fixed widths except Flags which is flex)
- Left/right arrows move cell cursor, NOT table scroll
- First column (#) is frozen and always visible when scrolling
- bubble-table provides "<" and ">" scroll indicators automatically

#### Navigation Keys
- **up/down**: Row navigation (handled by bubble-table)
- **left/right**: Cell navigation (handled by us, NOT passed to bubble-table)
- **c/o/g/e**: Set file disposition (commit/omit/gitignore/gitexclude)

---

## Two-Level Selection System

### The Challenge

We wanted both **row-level selection** (like traditional file managers) and **cell-level highlighting** (like spreadsheets) in the same table.

### The Solution

We implement a two-tier highlighting system:

1. **Row-level highlighting**
   - Entire selected row gets bold text
   - Entire selected row gets brighter colors (via HSL lightness adjustment)
   - Indicates which file is currently selected for operations

2. **Cell-level highlighting**
   - ONE cell within the selected row gets reverse-video
   - This is the "cursor" showing which column the user is focused on
   - Left/right arrows move this cursor

### Visual Example

```
Row 1 (unselected): dim colors, not bold
Row 2 (selected):   bright colors, bold, cell 3 has reverse-video ◄── current cell
Row 3 (unselected): dim colors, not bold
```

### Why This Approach?

- **Preserves semantic colors**: Disposition colors (green, yellow, red) remain recognizable
- **Clear visual hierarchy**: Selected row "pops" with brightness and boldness
- **Precise cursor**: Reverse-video shows exactly which cell can be edited/inspected
- **Excel-like UX**: Familiar to spreadsheet users

---

## Width Calculation Challenges

### The Problem

The table was consistently 4-10 characters too short, not reaching the right edge of the screen.

### Root Cause: Lipgloss Version Mismatch

```
Project dependency:     lipgloss v1.1.0
bubble-table uses:      lipgloss v0.5.0 (internally)
```

Border and padding calculations differ between these versions, causing layout discrepancies.

### The Solution

In `file_disposition_layout.go`:

```go
func (l FileDispositionModel) RightPaneInnerWidth() int {
    // Empirically determined: +2 offset needed due to lipgloss version mismatch
    return l.terminalWidth - l.leftPaneWidth + 2
}
```

This **+2 offset** is empirically determined (tested against actual rendering) rather than theoretically calculated.

### When This Might Break

- When bubble-table is updated to use lipgloss v2
- When we upgrade/downgrade lipgloss versions
- When terminal emulator changes border rendering

**Solution**: Test table width visually after any dependency updates. Adjust the offset if needed.

### Initialization Width Challenge

**Problem**: Table is created during file loading, which happens asynchronously and may occur before `WindowSizeMsg` arrives. Additionally, the tree width isn't known until the tree is created, but we need the tree width to calculate the table width.

**Root Cause**: Chicken-and-egg problem:
1. Layout calculations need tree width to determine right pane width
2. Tree hasn't been created yet during initial table creation
3. Terminal dimensions might not be available (WindowSizeMsg hasn't arrived)

**Solution** (in `editor_state.go`):

1. **`Layout()` method uses default tree width**:
```go
func (es EditorState) Layout() FileDispositionModel {
    const defaultTreeWidth = 40 // Reasonable default before tree is created

    treeWidth := defaultTreeWidth
    if es.ViewportsReady {
        treeWidth = es.FolderTree.LayoutWidth()  // Use actual width once tree exists
    }
    return NewFileDispositionModel(es.TerminalWidth(), es.TerminalHeight(), treeWidth)
}
```

2. **`TerminalWidth()` and `TerminalHeight()` methods query OS**:
```go
func (es EditorState) TerminalWidth() int {
    if es.Width > 0 {
        return es.Width  // Use WindowSizeMsg value if available
    }
    // Query actual terminal from OS if WindowSizeMsg hasn't arrived
    width, _, err := term.GetSize(int(os.Stdout.Fd()))
    if err != nil || width == 0 {
        return 120  // Reasonable fallback
    }
    return width
}
```

**Why This Works**:
- During initialization: terminalWidth=122 (from OS), treeWidth=40 (default) → RightPaneInnerWidth=84
- After tree created: terminalWidth=122 (from WindowSizeMsg), treeWidth=~40 (actual) → RightPaneInnerWidth=84
- Table created with correct width even before WindowSizeMsg arrives
- SetSize() updates table when terminal is resized or tree width changes

### Why Not Theoretical Calculation?

Theoretical calculation based on border widths:
```
Left pane:  leftPaneWidth + 4 (2 borders + 2 padding)
Right pane: tableWidth + 2 (2 borders)
Total:      (leftPaneWidth + 4) + (tableWidth + 2) = terminalWidth
Therefore:  tableWidth = terminalWidth - leftPaneWidth - 6
```

This gives `-6`, but actual rendering requires `+2` (an 8-character difference!). This large discrepancy confirms the lipgloss version mismatch theory.

---

## Styling Strategy

### Overview

We **disable** bubble-table's built-in row highlighting and implement our own cell-level styling.

### Why Disable Built-In Highlighting?

bubble-table's `HighlightStyle` applies a single style to the entire selected row:
- Can't apply different styles to different cells in the same row
- Can't apply reverse-video to just one cell
- Can't preserve semantic colors (disposition colors, change type colors)

### Our Approach

1. Set `HighlightStyle(lipgloss.NewStyle())` (empty style = disabled)
2. Apply styling at cell creation time in `buildTableRow()`
3. Each cell checks two flags:
   - `isSelectedRow`: Apply bold + brighter color to entire row
   - `isCurrentCell`: Also apply reverse-video to this one cell

### Color Adjustment with HSL

We use **HSL color space** (Hue, Saturation, Lightness) instead of RGB:

**Why HSL?**
- Allows adjusting brightness while preserving color identity
- Red stays red, just brighter or dimmer
- RGB adjustments would shift colors toward white/black

**Adjustments:**
- **Selected row**: Lightness increased by 40% (capped at 0.8)
- **Unselected row**: Lightness reduced to 60% (floored at 0.2)

**Example:**
```
COMMIT disposition (green #00FF00):
- Selected row:   #66FF66 (brighter green, bold)
- Unselected row: #009900 (dimmer green)
- Current cell:   #66FF66 with reverse-video (bright green inverted)
```

### Styled Cell Functions

Three functions create styled cells:

1. **`styledMetadataCell()`**: Generic cells (filename, size, permissions, etc.)
   - Uses base color (usually white)
   - Applies row/cell highlighting
   - Sets explicit background color for reverse-video visibility

2. **`styledDispositionCell()`**: Disposition column (COMMIT, OMIT, etc.)
   - Uses disposition's semantic color (green, yellow, red)
   - Applies row/cell highlighting
   - Preserves color meaning through brightness adjustment
   - Sets explicit background color for reverse-video visibility

3. **`styledChangeCell()`**: Change type column (M, A, D, U)
   - Uses change type's semantic color
   - Applies row/cell highlighting
   - Sets explicit background color for reverse-video visibility

**Important**: All styling functions call `.Background("#1a1a1a")` before `.Reverse(true)` to ensure reverse-video is visible. Without an explicit background, the reverse effect may not render properly in some terminals.

---

## Navigation Implementation

### Message Handling Strategy

The `Update()` method handles three types of interactions:

1. **Cell navigation** (left/right) - We handle AND pass to bubble-table
2. **Disposition keys** (c/o/g/e) - We handle, emit `DispositionChangedMsg`
3. **Row navigation** (up/down) - We delegate to bubble-table

### How We Handle Left/Right (Message Transformation)

We use **message transformation** to achieve both cell cursor movement AND horizontal scrolling:

1. **Always update** `currentColumn` on left/right keys
2. **Always rebuild** rows with new cell highlighting (reverse-video moves)
3. **Conditionally transform** messages before passing to bubble-table:
   - When table fits in viewport: Pass original left/right (bubble-table ignores)
   - When cell goes off-screen: Transform `left` → `shift+left`, `right` → `shift+right`
   - bubble-table scrolls on shift+left/shift+right (its default scroll keys)

**Why this works:**
- We removed custom keymap (which mapped left/right to ScrollLeft/ScrollRight)
- Using bubble-table's **default keybindings** where shift+arrows trigger scroll
- We conditionally inject shift modifier when scrolling is actually needed
- This prevents unwanted scrolling when table fits in viewport

**The check:**
```go
needsScroll := m.width > 0 && m.width < m.widthToColumn(m.currentColumn+1)
```

This gives us Excel-like behavior:
1. Left/right arrows move cell cursor within visible columns (no scroll)
2. When cell reaches edge: Transform to shift+left/shift+right
3. bubble-table receives shift modifier and scrolls horizontally
4. Cell cursor stays visible as table scrolls

### Row Rebuilding on Every Change

Every time selection changes (row OR column OR disposition), we call:
```go
m.table = m.table.WithRows(buildTableRows(...))
```

This regenerates **ALL row data** with updated styling.

**Why rebuild everything?**
- bubble-table doesn't expose cell-level styling hooks
- We need to apply `isSelectedRow`/`isCurrentCell` flags to each cell
- Can't just update one cell - must regenerate entire row
- Can't just update one row - other rows need dimming/brightening updates

**The rebuilding pattern:**
1. Get current selected row index from bubble-table
2. Call `buildTableRows()` with `selectedRowIndex` and `currentColumn`
3. Each row checks if it's selected
4. Each cell checks if it's selected AND in current column
5. Apply styling flags (bold for row, reverse-video for cell)
6. Replace table rows with newly styled versions

**Performance:**
- O(n) where n = number of files in directory
- For 10-100 files: imperceptible (<1ms)
- For >1000 files: may need optimization (rebuild only visible rows)

### Delegation to bubble-table

For row navigation (up/down), we:
1. Save current row index before delegating
2. Call `m.table.Update(msg)` to let bubble-table handle up/down
3. Check if row index changed
4. If changed, rebuild rows to update ▶ indicator, bold styling, and reverse-video

---

## Performance Considerations

### Current Performance

- **Small directories** (10-100 files): Excellent, imperceptible
- **Medium directories** (100-500 files): Good, minor lag on selection changes
- **Large directories** (500-1000 files): Acceptable, noticeable lag
- **Very large directories** (>1000 files): May need optimization

### Optimization Strategies (Future)

If performance becomes an issue:

1. **Lazy row building**: Only rebuild visible rows (requires tracking viewport offset)
2. **Memoization**: Cache styled cells, invalidate on selection change
3. **Incremental updates**: Only rebuild changed rows (complex, requires tracking previous state)
4. **Virtual scrolling**: Only render visible rows (requires deeper integration with bubble-table)

### Why We Haven't Optimized Yet

- Typical use case: directories with 10-100 files
- Current performance is acceptable for this scale
- Premature optimization would add complexity
- If needed, we can optimize later without changing the API

---

## Dependencies

### Direct Dependencies

- `github.com/evertras/bubble-table` v0.19.2
  - Uses lipgloss v0.5.0 internally
  - Provides table rendering and row navigation
  - We handle cell-level navigation ourselves

- `github.com/charmbracelet/lipgloss` v1.1.0
  - Project-wide dependency
  - Used for all styling (colors, borders, padding)
  - **Version mismatch with bubble-table causes width calculation issues**

- `github.com/lucasb-eyer/go-colorful`
  - HSL color space calculations
  - Used in `adjustColorForSelection()` for brightness adjustments

### Indirect Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - TUI components

---

## Future Considerations

### When bubble-table v2 is Released

The [bubble-table v2-exp branch](https://github.com/Evertras/bubble-table/issues/194) is being updated to work with bubbletea/lipgloss v2.

**When v2 is stable, consider:**
1. Updating to bubble-table v2
2. Re-testing width calculations (may need to adjust or remove the +2 offset)
3. Checking if new styling APIs make our approach cleaner
4. Evaluating if scroll indicators can be disabled/customized

### Potential Improvements

- **Custom scroll indicators**: Replace "<" and ">" with something more subtle
- **Keyboard shortcuts overlay**: Show available keys in a help popup
- **Column reordering**: Allow users to customize column order
- **Column resizing**: Allow users to adjust column widths
- **Column hiding**: Allow users to hide columns they don't care about
- **Sorting**: Click column header to sort by that column
- **Filtering**: Type to filter visible rows

These are **not planned** but worth considering if user feedback requests them.

---

## Testing Notes

### Manual Testing Checklist

When making changes to FilesTableModel:

- [ ] Table fills screen width (reaches right edge)
- [ ] Content view also fills screen width (comparison test)
- [ ] Left/right arrows move cell cursor, not just scroll
- [ ] Up/down arrows move row selection
- [ ] Selected row is bold with brighter colors
- [ ] Current cell has reverse-video
- [ ] Disposition colors are preserved (green=COMMIT, yellow=OMIT, etc.)
- [ ] First column (#) stays frozen when scrolling horizontally
- [ ] "<" and ">" scroll indicators appear when table is wider than screen
- [ ] Disposition keys (c/o/g/e) work on selected file
- [ ] Table resizes correctly on terminal resize

### Width Calculation Testing

To test width calculations after dependency updates:

1. Build and run gomion
2. Navigate to a directory view (table visible)
3. **Narrow terminal** (< 100 columns): Table should fill width, no gap on right
4. **Wide terminal** (> 200 columns): Table should fill width, no gap on right
5. **Compare with file content view**: Both should reach right edge equally

If table is too short/long:
- Adjust the offset in `RightPaneInnerWidth()` in `file_disposition_layout.go`
- Test both narrow and wide terminals
- Document the new offset value

---

## Code Organization

### Main Files

- **`files_table_model.go`**: Table model implementation
  - Column definitions
  - Row building with styling
  - Cell navigation (left/right handling)
  - Two-level highlighting system

- **`file_disposition_layout.go`**: Layout calculations
  - Width calculations (with lipgloss version mismatch handling)
  - Height calculations
  - Pane sizing for left (tree) and right (table/content)

- **`editor_state.go`**: Main TUI state
  - Manages file selection view
  - Coordinates between tree, table, and content panes
  - Handles focus and view switching

### Supporting Files

- **`file_disposition_node.go`**: Tree node implementation
- **`file_disposition_tree_model.go`**: Tree view for file hierarchy
- **`file_content_model.go`**: File content viewer (right pane when file selected)
- **`directory.go`**: Directory model with file list
- **`file.go`**: File model with metadata
- **`file_disposition.go`**: Disposition enum (COMMIT, OMIT, IGNORE, EXCLUDE)

---

## See Also

- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [bubble-table Documentation](https://github.com/Evertras/bubble-table)
- [lipgloss Documentation](https://github.com/charmbracelet/lipgloss)
- [go-colorful Documentation](https://github.com/lucasb-eyer/go-colorful)
