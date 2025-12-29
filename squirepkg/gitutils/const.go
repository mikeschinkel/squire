package gitutils

import (
	"os"

	"github.com/mikeschinkel/go-dt"
)

const IgnoreFilename dt.Filename = ".gitignore"
const RepoPath dt.PathSegment = ".git"
const KeepFile dt.PathSegment = ".gitkeep"

var InfoPath = dt.PathSegmentsJoin(RepoPath, "info")
var ExcludeFilepath = dt.RelFilepathJoin(InfoPath, "exclude")

type IgnoreFile struct {
	ConfigFile
}

func NewIgnoreFile(baseDir dt.DirPath) *IgnoreFile {
	return &IgnoreFile{
		ConfigFile: newConfigFile(baseDir, IgnoreFilename),
	}
}

type ExcludeFile struct {
	ConfigFile
}

func NewExcludeFile(baseDir dt.DirPath) *ExcludeFile {
	return &ExcludeFile{
		ConfigFile: newConfigFile(baseDir, ExcludeFilepath),
	}
}

type ConfigFile struct {
	filePath dt.Filepath
}

type configFileable interface {
	dt.Filename | dt.RelFilepath
}

func newConfigFile[CF configFileable](baseDir dt.DirPath, fn CF) ConfigFile {
	return ConfigFile{filePath: dt.FilepathJoin(baseDir, fn)}
}
func (cf ConfigFile) EnsureFile() (ef dt.Filepath, err error) {
	err = cf.dir().MkdirAll(0755)
	if err != nil {
		err = NewErr(dt.ErrFailedtoCreateDir, cf.dir().ErrKV(), err)
	}
	return ef, err
}
func (cf ConfigFile) AppendFilename(file dt.Filename) (err error) {
	return cf.AppendLine(string(file))
}
func (cf ConfigFile) AppendPathSegment(ps dt.PathSegment) (err error) {
	return cf.AppendLine(string(ps + "/"))
}
func (cf ConfigFile) AppendLine(line string) (err error) {
	var f *os.File
	f, err = cf.filePath.OpenFile(os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		err = NewErr(dt.ErrFailedToOpenFile, err)
		goto end
	}
	_, err = f.WriteString(line + "\n")
	defer dt.CloseOrLog(f)
	if err != nil {
		err = NewErr(dt.ErrFailedToWriteToFile, err)
		goto end
	}
end:
	if err != nil {
		err = WithErr(err, cf.filePath.ErrKV())
	}
	return err
}

func (cf ConfigFile) ContainsFilename(file dt.Filename) (contains bool, err error) {
	return cf.containsPath(dt.EntryPath(file))
}

func (cf ConfigFile) ContainsPathSegment(ps dt.PathSegment) (contains bool, err error) {
	return cf.containsPath(dt.EntryPath(ps + "/"))
}

func (cf ConfigFile) containsPath(path dt.EntryPath) (contains bool, err error) {
	var contents []byte

	// Read existing file if it exists
	contents, err = cf.filePath.ReadFile()
	if os.IsNotExist(err) {
		err = nil
		goto end
	}
	if err != nil {
		err = NewErr(dt.ErrFailedToCopyFile, cf.filePath.ErrKV(), err)
		goto end
	}

	// Check if path is in file
	contains = cf.containsLine(contents, string(path))
end:
	return contains, err
}

// dir returns the directory containing of the ConfigFile
func (cf ConfigFile) dir() dt.DirPath {
	return cf.filePath.Dir()
}

// containsLine checks if content contains a specific line
func (cf ConfigFile) containsLine(content []byte, line string) bool {
	lines := string(content)
	needle := line + "\n"
	return cf.containsSubstr(lines, needle) || cf.containsSubstr(lines, line)
}

// contains checks if s contains substr
func (cf ConfigFile) containsSubstr(s, substr string) (contains bool) {
	if len(s) < len(substr) {
		goto end
	}
	if s == substr {
		contains = true
		goto end
	}
	if cf.findSubstr(s, substr) == -1 {
		goto end
	}
	contains = true
end:
	return contains
}

// findSubstr returns the index of substr in s, or -1 if not found
func (cf ConfigFile) findSubstr(s, substr string) (index int) {
	index = -1
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] != substr {
			continue
		}
		index = i
		goto end
	}
end:
	return index
}
