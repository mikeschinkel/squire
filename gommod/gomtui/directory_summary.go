package gomtui

import (
	"github.com/mikeschinkel/gomion/gommod/gitutils"
)

// calculateDirSummary calculates summary statistics for a directory's files.
// Iterates through files and counts by disposition, git status, and change type.
// Sums total size from file metadata.
func calculateDirSummary(files []*File) (summary DirSummary) {
	// Initialize summary
	summary.TotalFiles = len(files)

	for _, file := range files {
		// Count by disposition
		switch file.Disposition {
		case CommitDisposition:
			summary.CommitCount++
		case OmitDisposition:
			summary.OmitCount++
		case GitIgnoreDisposition:
			summary.GitIgnoreCount++
		case GitExcludeDisposition:
			summary.GitExcludeCount++
		}

		// Skip files without metadata
		if file.Metadata == nil {
			continue
		}

		// Sum file size
		summary.TotalSize += file.Metadata.Size

		// Count by git status (staged/unstaged)
		switch file.Metadata.Staging {
		case gitutils.IndexStaging:
			summary.StagedCount++
		case gitutils.WorktreeStaging:
			summary.UnstagedCount++
		case gitutils.BothStaging:
			summary.StagedCount++
			summary.UnstagedCount++
		}

		// Count by change type (use unstaged if available, otherwise staged)
		changeType := file.Metadata.UnstagedChange
		if changeType == gitutils.UnknownChangeType {
			changeType = file.Metadata.StagedChange
		}

		switch changeType {
		case gitutils.ModifiedChangeType:
			summary.ModifiedCount++
		case gitutils.AddedChangeType:
			summary.AddedCount++
		case gitutils.DeletedChangeType:
			summary.DeletedCount++
		case gitutils.RenamedChangeType:
			summary.RenamedCount++
		case gitutils.UntrackedChangeType:
			summary.UntrackedCount++
		}
	}

	return summary
}
