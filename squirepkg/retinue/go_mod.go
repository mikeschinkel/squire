package retinue

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

type GoMod struct {
	FilePath dt.Filepath
	Logger   *slog.Logger
}

func NewGoMod(filePath dt.Filepath) *GoMod {
	return &GoMod{FilePath: filePath}
}

// Read reads the input file and returns a list of go.mod paths
func (gm GoMod) Read() (paths []dt.Filepath, err error) {
	var inputPath dt.Filepath
	var file *os.File
	var scanner *bufio.Scanner
	var lineNum int
	var line string
	var goModPath dt.Filepath
	var exists bool

	// Parse and expand the input file path
	inputPath, err = dt.ParseFilepath(string(gm.FilePath))
	if err != nil {
		err = fmt.Errorf("parsing file path %q: %w", gm.FilePath, err)
		goto end
	}

	file, err = inputPath.Open()
	if err != nil {
		err = fmt.Errorf("opening file %s: %w", inputPath, err)
		goto end
	}
	defer dt.CloseOrLog(file)

	scanner = bufio.NewScanner(file)
	lineNum = 0

	for scanner.Scan() {
		lineNum++
		line = strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			continue
		}

		// Parse and expand go.mod path
		goModPath, err = dt.ParseFilepath(line)
		if err != nil {
			gm.Logger.Warn("failed to parse path", "line", lineNum, "path", line, "error", err)
			err = nil
			continue
		}

		// Verify file exists
		exists, err = goModPath.Exists()
		if err != nil {
			gm.Logger.Warn("error checking file existence", "line", lineNum, "path", goModPath, "error", err)
			err = nil
			continue
		}

		if !exists {
			gm.Logger.Warn("go.mod file does not exist", "line", lineNum, "path", goModPath)
			continue
		}

		paths = append(paths, goModPath)
	}

	err = scanner.Err()
	if err != nil {
		err = fmt.Errorf("reading file %s: %w", inputPath, err)
		goto end
	}

end:
	return paths, err
}
