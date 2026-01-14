package gomtui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
)

// ============================================================================
// TODO: Move these "Helper" functions to files where the yare related to the
//  use-case, and maybe make them methods.
// ============================================================================

// extractRenamedFilePath extracts the actual path from a renamed file pattern.
// Git shows renamed files as "oldpath -> newpath". This function returns the new path.
// If the input is not a rename pattern, it returns the input unchanged.
func extractRenamedFilePath(path dt.RelFilepath) dt.RelFilepath {
	var parts []string
	if !strings.Contains(string(path), " -> ") {
		goto end
	}
	parts = strings.Split(string(path), " -> ")
	path = dt.RelFilepath(strings.TrimSpace(parts[1]))
end:
	return path
}

// ============================================================================
// Business Logic Functions
// These functions contain the actual I/O and business logic,
// called by Cmd functions. They are testable and have clear names.
// ============================================================================

// loadGitStatus loads git status for a repository
func loadGitStatus(userRepo *gitutils.Repo) (gitutils.StatusMap, error) {
	return userRepo.StatusMap(context.Background(), &gitutils.StatusArgs{
		HumanReadable: false,
	})
}

// loadFileContent loads file content with size checking and truncation
func loadFileContent(repoRoot dt.DirPath, relPath dt.RelFilepath) (content string, yOffset int, err error) {
	var info os.FileInfo
	var fp dt.Filepath

	fp = dt.FilepathJoin(repoRoot, relPath)

	// Check file size before loading
	info, err = fp.Stat()
	if err != nil {
		goto end
	}

	// Determine loading strategy based on file size
	switch {
	case info.Size() <= MaxFileContentSize:
		// Small file - load entire content
		content, err = readFileContent(repoRoot, relPath, 0)
		if err != nil {
			goto end
		}

	case info.Size() <= MaxFilePreviewSize:
		// Medium file (32-128KB) - load entire content with warning
		content, err = readFileContent(repoRoot, relPath, 0)
		if err != nil {
			goto end
		}

	default:
		// Large file (>128KB) - load first 128KB with warning
		content, err = readFileContent(repoRoot, relPath, MaxFilePreviewSize)
		if err != nil {
			goto end
		}
		// Prepend red warning header
		warning := lipgloss.NewStyle().
			Foreground(lipgloss.Color(RedColor)).
			Bold(true).
			Render(fmt.Sprintf("⚠ LARGE FILE: %d KB (showing first 128 KB only)\n\n", info.Size()/1024))
		content = warning + content + "\n\n[... truncated ...]"
	}

end:
	if err != nil {
		err = WithErr(err, ErrLoadingFile, fp.ErrKV())
	}
	return content, yOffset, err
}

// readFileContent reads file content handling renamed files
// maxBytes: 0 or -1 = read entire file, otherwise read first N bytes
func readFileContent(repoRoot dt.DirPath, relPath dt.RelFilepath, maxBytes int64) (content string, err error) {
	var fp dt.Filepath
	var bytes []byte

	// Handle renamed files (format: "oldpath -> newpath")
	relPath = extractRenamedFilePath(relPath)

	// Construct full path
	fp = dt.FilepathJoin(repoRoot, relPath)

	// Determine read strategy
	switch maxBytes {
	case 0, -1:
		// Read entire file
		bytes, err = fp.ReadFile()
		if err != nil {
			err = NewErr(ErrGit, fp.ErrKV(), err)
			goto end
		}
	default:
		// Read partial file
		var file *os.File
		file, err = fp.Open()
		if err != nil {
			err = NewErr(ErrGit, fp.ErrKV(), err)
			goto end
		}
		defer dt.CloseOrLog(file)

		buf := make([]byte, maxBytes)
		n, readErr := file.Read(buf)
		if readErr != nil && readErr.Error() != "EOF" {
			err = NewErr(ErrGit, fp.ErrKV(), readErr)
			goto end
		}
		bytes = buf[:n]
	}

	content = string(bytes)

end:
	return content, err
}
