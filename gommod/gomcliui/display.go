package gomcliui

import (
	"strings"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/gomion/gommod/precommit"
)

// DisplayBox draws a box around content with a title
func DisplayBox(title string, content string, w cliutil.Writer) {
	width := 60
	if len(title) > width-4 {
		width = len(title) + 4
	}

	// Top border with title
	w.Printf("┌─ %s %s┐\n", title, strings.Repeat("─", width-len(title)-4))

	// Content lines
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		w.Printf("│ %-*s │\n", width-4, line)
	}

	// Bottom border
	w.Printf("└%s┘\n", strings.Repeat("─", width-2))
}

// DisplayTable displays data in a simple table format
func DisplayTable(headers []string, rows [][]string, w cliutil.Writer) {
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
		w.Printf("%-*s  ", colWidths[i], header)
	}
	w.Printf("\n")

	// Display separator
	for _, width := range colWidths {
		w.Printf("%s  ", strings.Repeat("─", width))
	}
	w.Printf("\n")

	// Display rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				w.Printf("%-*s  ", colWidths[i], cell)
			}
		}
		w.Printf("\n")
	}
}

// DisplayProgress shows a progress message
func DisplayProgress(message string, w cliutil.Writer) {
	w.Printf("⏳ %s...\n", message)
}

// DisplaySuccess shows a success message
func DisplaySuccess(message string, w cliutil.Writer) {
	w.Printf("✓ %s\n", message)
}

// DisplayError shows an error message
func DisplayError(message string, w cliutil.Writer) {
	w.Printf("✗ %s\n", message)
}

// DisplayWarning shows a warning message
func DisplayWarning(message string, w cliutil.Writer) {
	w.Printf("⚠ %s\n", message)
}

// DisplaySeparator displays a visual separator line
func DisplaySeparator(w cliutil.Writer) {
	w.Printf("%s\n", strings.Repeat("─", 60))
}

// DisplayAnalysisReport shows detailed pre-commit analysis results
func DisplayAnalysisReport(results *precommit.Results, writer cliutil.Writer) {
	writer.Printf("\n")
	writer.Printf("═══════════════════════════════════════\n")
	writer.Printf("  PRE-COMMIT ANALYSIS REPORT\n")
	writer.Printf("═══════════════════════════════════════\n\n")

	// Display formatted analysis using markdown format for readability
	report := results.FormatForAI()
	writer.Printf("%s\n", report)

	writer.Printf("═══════════════════════════════════════\n\n")
	writer.Printf("Press any key to continue...")
	_, _ = cliutil.ReadSingleKey()
	writer.Printf("\n\n")
}

// DisplayCommitGroups shows suggested commit groupings
func DisplayCommitGroups(groups []precommit.CommitGroup, writer cliutil.Writer) {
	if len(groups) == 0 {
		writer.Printf("No staged files to group.\n")
		return
	}

	writer.Printf("\nAnalyzing staged changes for commit grouping...\n\n")
	writer.Printf("Suggested commit groups:\n\n")

	for i, group := range groups {
		writer.Printf("%d. %s\n", i+1, group.Title)
		writer.Printf("   Rationale: %s\n", group.Rationale)
		writer.Printf("   Files (%d):\n", len(group.Files))
		for _, file := range group.Files {
			writer.Printf("     - %s\n", file)
		}
		writer.Printf("\n")
	}

	// TODO: Implement interactive restaging
	writer.Printf("Interactive restaging not yet implemented.\n")
	writer.Printf("For now, you can manually stage and commit these groups.\n\n")
}
