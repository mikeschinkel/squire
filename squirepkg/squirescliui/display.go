package squirescliui

import (
	"fmt"
	"io"
	"strings"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/squire/squirepkg/precommit"
)

// DisplayBox draws a box around content with a title
func DisplayBox(title string, content string, w io.Writer) {
	width := 60
	if len(title) > width-4 {
		width = len(title) + 4
	}

	// Top border with title
	fmt.Fprintf(w, "┌─ %s %s┐\n", title, strings.Repeat("─", width-len(title)-4))

	// Content lines
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		fmt.Fprintf(w, "│ %-*s │\n", width-4, line)
	}

	// Bottom border
	fmt.Fprintf(w, "└%s┘\n", strings.Repeat("─", width-2))
}

// DisplayTable displays data in a simple table format
func DisplayTable(headers []string, rows [][]string, w io.Writer) {
	if len(headers) == 0 {
		return
	}

	// Calculate column widths
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Display headers
	for i, header := range headers {
		fmt.Fprintf(w, "%-*s  ", colWidths[i], header)
	}
	fmt.Fprintln(w)

	// Display separator
	for _, width := range colWidths {
		fmt.Fprintf(w, "%s  ", strings.Repeat("─", width))
	}
	fmt.Fprintln(w)

	// Display rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				fmt.Fprintf(w, "%-*s  ", colWidths[i], cell)
			}
		}
		fmt.Fprintln(w)
	}
}

// DisplayProgress shows a progress message
func DisplayProgress(message string, w io.Writer) {
	fmt.Fprintf(w, "⏳ %s...\n", message)
}

// DisplaySuccess shows a success message
func DisplaySuccess(message string, w io.Writer) {
	fmt.Fprintf(w, "✓ %s\n", message)
}

// DisplayError shows an error message
func DisplayError(message string, w io.Writer) {
	fmt.Fprintf(w, "✗ %s\n", message)
}

// DisplayWarning shows a warning message
func DisplayWarning(message string, w io.Writer) {
	fmt.Fprintf(w, "⚠ %s\n", message)
}

// DisplaySeparator displays a visual separator line
func DisplaySeparator(w io.Writer) {
	fmt.Fprintln(w, strings.Repeat("─", 60))
}

// DisplayAnalysisReport shows detailed pre-commit analysis results
func DisplayAnalysisReport(results *precommit.Results, writer io.Writer) {
	dtx.Fprintf(writer, "\n")
	dtx.Fprintf(writer, "═══════════════════════════════════════\n")
	dtx.Fprintf(writer, "  PRE-COMMIT ANALYSIS REPORT\n")
	dtx.Fprintf(writer, "═══════════════════════════════════════\n\n")

	// Display formatted analysis using markdown format for readability
	report := results.FormatForAI()
	dtx.Fprintf(writer, "%s\n", report)

	dtx.Fprintf(writer, "═══════════════════════════════════════\n\n")
	dtx.Fprintf(writer, "Press any key to continue...")
	_, _ = cliutil.ReadSingleKey()
	dtx.Fprintf(writer, "\n\n")
}

// DisplayCommitGroups shows suggested commit groupings
func DisplayCommitGroups(groups []precommit.CommitGroup, writer io.Writer) {
	if len(groups) == 0 {
		dtx.Fprintf(writer, "No staged files to group.\n")
		return
	}

	dtx.Fprintf(writer, "\nAnalyzing staged changes for commit grouping...\n\n")
	dtx.Fprintf(writer, "Suggested commit groups:\n\n")

	for i, group := range groups {
		dtx.Fprintf(writer, "%d. %s\n", i+1, group.Title)
		dtx.Fprintf(writer, "   Rationale: %s\n", group.Rationale)
		dtx.Fprintf(writer, "   Files (%d):\n", len(group.Files))
		for _, file := range group.Files {
			dtx.Fprintf(writer, "     - %s\n", file)
		}
		dtx.Fprintf(writer, "\n")
	}

	// TODO: Implement interactive restaging
	dtx.Fprintf(writer, "Interactive restaging not yet implemented.\n")
	dtx.Fprintf(writer, "For now, you can manually stage and commit these groups.\n\n")
}
