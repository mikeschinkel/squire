package gomtui

import (
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/bubbletree"
)

// FileDispositionTree wraps bubbletree. for hierarchical file display
type FileDispositionTree struct {
	tree *bubbletree.Tree[bubbletree.File]
}

// NewFileDispositionTree creates a new folder tree model from a flat list of files
func NewFileDispositionTree(fileSource *FileSource, dispositionFn func(dt.RelFilepath) FileDisposition) (m FileDispositionTree) {
	// Build tree hierarchy
	//rootNodes := m.buildTree(fileSource.Files(), "")
	//rootNodes := m.buildTree(fileSource.Files(), "")

	fileTree := bubbletree.FileTree{}

	files := fileTree.BuildTree(fileSource.Files())
	nodes := make([]*bubbletree.FileNode, len(files))
	for i, file := range files {
		nodes[i] = file
	}
	// Create tree with custom provider
	tree := bubbletree.NewTree[bubbletree.File](nodes, &bubbletree.TreeArgs[bubbletree.File]{
		NodeProvider: NewFileDispositionNodeProvider(dispositionFn),
	})

	return FileDispositionTree{tree: tree}
}

// NewEmptyFileDispositionTree creates an empty tree with a message
func NewEmptyFileDispositionTree(message string, width, height int, dispositionFn func(dt.RelFilepath) FileDisposition) FileDispositionTree {
	// Create a single node with the message
	nodes := []*bubbletree.FileNode{
		bubbletree.NewNode("empty", message, bubbletree.File{
			Path: dt.RelFilepath(""),
		}),
	}
	tree := bubbletree.NewTree[bubbletree.File](nodes, &bubbletree.TreeArgs[bubbletree.File]{
		NodeProvider: NewFileDispositionNodeProvider(dispositionFn),
	})

	return FileDispositionTree{tree: tree}
}

func (m FileDispositionTree) HasTree() bool {
	return m.tree != nil
}

// SelectedFile returns the currently selected file/folder
func (m FileDispositionTree) SelectedFile() *bubbletree.File {
	node := m.tree.FocusedNode()
	if node == nil {
		return nil
	}
	return node.Data()
}

// SelectedNode returns the currently selected node
func (m FileDispositionTree) SelectedNode() *bubbletree.FileNode {
	return m.tree.FocusedNode()
}
