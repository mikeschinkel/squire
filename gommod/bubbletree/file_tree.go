package bubbletree

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/mikeschinkel/go-dt"
)

type File struct {
	Path    dt.RelFilepath
	Content string
	data    any
	YOffset int // Viewport scroll position

	// Cached file metadata (for directory table display)
	meta *FileMeta // nil if not yet loaded
}

func NewFile(path dt.RelFilepath, content string) *File {
	return &File{
		Path:    path,
		Content: content,
	}
}

// FileMeta contains cached meta about a file for display in directory tables.
type FileMeta struct {
	Size        int64          // File size in bytes
	ModTime     time.Time      // Modification time
	Permissions os.FileMode    // Full permissions
	EntryStatus dt.EntryStatus // File, Dir, Symlink, etc.
	Data        any
}

type FileNode = Node[File]
type FileTree Tree[File]

func (f *File) IsEmpty() bool {
	if f == nil {
		return true
	}
	return f.Path == ""
}

func (f *File) Data() any {
	if f.data == nil {
		panic("File.Data() called before ....???")
	}
	return f.data
}

func (f *File) HasData() bool {
	return f.data != nil
}

func (f *File) SetData(data any) *File {
	f.data = data
	return f
}

func (f *File) Meta() *FileMeta {
	if f.meta == nil {
		panic("File.Meta() called before File.LoadMeta()")
	}
	return f.meta
}

func (f *File) HasMeta() bool {
	return f.meta != nil
}

func (f *File) SetMeta(meta *FileMeta) {
	f.meta = meta
}

func (f *File) LoadMeta(root dt.DirPath) (err error) {
	var fp dt.Filepath
	var info os.FileInfo

	// Initialize metadata if not already present
	if f.meta != nil {
		goto end
	}
	// Construct full path
	fp = dt.FilepathJoin(root, f.Path)

	// Get f info using dt.Filepath.Stat() method
	info, err = fp.Stat()
	if err != nil {
		if os.IsNotExist(err) {
			// File might be deleted after git status - return nil (not an error)
			f.meta = &FileMeta{}
			err = nil
			goto end
		}
		err = NewErr(dt.ErrFileSystem, dt.ErrFileStat, fp.ErrKV(), err)
		goto end
	}

	// Initialize metadata if not already present
	f.meta = &FileMeta{
		Size:        info.Size(),
		ModTime:     info.ModTime(),
		Permissions: info.Mode(),
		EntryStatus: dt.GetEntryStatus(info),
	}
end:
	return err
}

// BuildTree creates a hierarchical tree from flat file list
// Returns the top-level nodes directly (no wrapper root) to save indentation
func (m *FileTree) BuildTree(files []File) []*FileNode {
	// Create temporary root for building, but we'll return its children
	root := NewNode(".", ".", File{
		Path: ".",
	})

	// Sort files by path first - this allows efficient tree building
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	// Build tree structure using path-based node map for O(1) lookups
	nodeMap := make(map[string]*FileNode, len(files)-1)
	nodeMap["."] = root

	for _, file := range files {
		path := file.Path
		if path == "" {
			continue
		}
		segments := path.Split("/")

		currentPath := ""

		// Create folder nodes for each segment (except last, which is the file)
		for i := 0; i < len(segments)-1; i++ {
			segment := segments[i]
			switch {
			case currentPath == "":
				currentPath = string(segment)
			default:
				currentPath = currentPath + "/" + string(segment)
			}

			// Check if this folder node already exists
			_, exists := nodeMap[currentPath]
			if !exists {
				// Create new folder node (id=fullPath, name=basename)
				folderNode := NewNode(
					currentPath,                // id
					filepath.Base(currentPath), // name (basename for display)
					File{
						Path: dt.RelFilepath(currentPath),
					},
				)

				// Find parent node
				parentPath := filepath.Dir(currentPath)
				if parentPath == "" {
					parentPath = "."
				}
				parentNode := nodeMap[parentPath]

				// Add to parent
				parentNode.AddChild(folderNode)

				// Add to map
				nodeMap[currentPath] = folderNode
			}
		}

		// Add file node (id=fullPath, name=basename)
		fileNode := NewNode(
			string(path),        // id
			string(path.Base()), // name (basename for display)
			file,
		)

		// Collapse all folders (first level should be visible but collapsed)
		fileNode.expanded = false

		// Find parent node for this file
		parentPath := path.Dir()
		if parentPath == "" {
			parentPath = "."
		}
		parentNode := nodeMap[string(parentPath)]

		// Add file to parent folder
		parentNode.AddChild(fileNode)
	}

	//// Collapse all folders (first level should be visible but collapsed)
	//for _, child := range root.Children() {
	//	m.collapseAll(child)
	//}

	// Return children directly (skip the temporary root to save indentation)
	return root.Children()
}
