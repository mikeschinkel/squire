package gomtui

import (
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/bubbletree"
)

// FileDispositionTreeModel wraps bubbletree.Model for hierarchical file display
type FileDispositionTreeModel struct {
	model bubbletree.Model[File]
}

// NewFileDispositionTreeModel creates a new folder tree model from a flat list of files
func NewFileDispositionTreeModel(files []File, width, height int) FileDispositionTreeModel {
	// Build tree hierarchy
	rootNodes := buildFileDispositionTreeHierarchy(files)

	// Create tree with custom provider
	tree := bubbletree.NewTree[File](rootNodes, &bubbletree.TreeArgs[File]{
		NodeProvider: NewFileDispositionNodeProvider(),
	})

	// Create BubbleTea model
	model := bubbletree.NewModel(tree, width, height)

	return FileDispositionTreeModel{
		model: model,
	}
}

// NewEmptyFileDispositionTreeModel creates an empty tree with a message
func NewEmptyFileDispositionTreeModel(message string, width, height int) FileDispositionTreeModel {
	// Create a single node with the message
	nodes := []*FileDispositionNode{
		bubbletree.NewNode("empty", message, File{
			Path:        dt.RelFilepath(""),
			Disposition: UnspecifiedDisposition,
			Content:     "",
		}),
	}

	tree := bubbletree.NewTree[File](nodes, &bubbletree.TreeArgs[File]{
		NodeProvider: NewFileDispositionNodeProvider(),
	})

	model := bubbletree.NewModel(tree, width, height)

	return FileDispositionTreeModel{
		model: model,
	}
}

// Init initializes the model
func (m FileDispositionTreeModel) Init() tea.Cmd {
	return m.model.Init()
}

// Update handles messages and updates the model
func (m FileDispositionTreeModel) Update(msg tea.Msg) (FileDispositionTreeModel, tea.Cmd) {
	updatedModel, cmd := m.model.Update(msg)
	m.model = updatedModel
	return m, cmd
}

// View renders the tree
func (m FileDispositionTreeModel) View() string {
	return m.model.View()
}

// SelectedFile returns the currently selected file/folder
func (m FileDispositionTreeModel) SelectedFile() *File {
	node := m.model.FocusedNode()
	if node == nil {
		return nil
	}
	return node.Data()
}

// SelectedNode returns the currently selected node
func (m FileDispositionTreeModel) SelectedNode() *FileDispositionNode {
	return m.model.FocusedNode()
}

// SetSize updates the model dimensions
func (m FileDispositionTreeModel) SetSize(width, height int) FileDispositionTreeModel {
	m.model = m.model.SetSize(width, height)
	return m
}

// MaxVisibleWidth calculates the actual width of the longest rendered line
func (m FileDispositionTreeModel) MaxVisibleWidth() int {
	return m.model.MaxLineWidth()
}

// LayoutWidth returns the width this component needs for layout purposes.
func (m FileDispositionTreeModel) LayoutWidth() int {
	return m.MaxVisibleWidth()
}

// buildFileDispositionTreeHierarchy creates a hierarchical tree from flat file list
// Returns the top-level nodes directly (no wrapper root) to save indentation
func buildFileDispositionTreeHierarchy(files []File) []*FileDispositionNode {
	// Create temporary root for building, but we'll return its children
	root := bubbletree.NewNode(".", ".", File{
		Path:        dt.RelFilepath("."),
		Disposition: CommitDisposition,
		Content:     "",
	})

	// Sort files by path first - this allows efficient tree building
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	// Build tree structure using path-based node map for O(1) lookups
	nodeMap := make(map[string]*FileDispositionNode)
	nodeMap["."] = root

	for _, file := range files {
		pathStr := string(file.Path)
		segments := strings.Split(pathStr, "/")

		currentPath := ""

		// Create folder nodes for each segment (except last, which is the file)
		for i := 0; i < len(segments)-1; i++ {
			segment := segments[i]
			if currentPath == "" {
				currentPath = segment
			} else {
				currentPath = currentPath + "/" + segment
			}

			// Check if this folder node already exists
			if _, exists := nodeMap[currentPath]; !exists {
				// Create new folder node (id=fullPath, name=basename)
				folderNode := bubbletree.NewNode(
					currentPath,                // id
					filepath.Base(currentPath), // name (basename for display)
					File{
						Path:        dt.RelFilepath(currentPath),
						Disposition: CommitDisposition,
						Content:     "",
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
		fileNode := bubbletree.NewNode(
			pathStr,                // id
			filepath.Base(pathStr), // name (basename for display)
			file,
		)

		// Find parent node for this file
		parentPath := filepath.Dir(pathStr)
		if parentPath == "" {
			parentPath = "."
		}
		parentNode := nodeMap[parentPath]

		// Add file to parent folder
		parentNode.AddChild(fileNode)
	}

	// Collapse all folders (first level should be visible but collapsed)
	for _, child := range root.Children() {
		collapseAllFileDispositionNodes(child)
	}

	// Return children directly (skip the temporary root to save indentation)
	return root.Children()
}

// collapseAllBubbletree recursively collapses all nodes in the tree
func collapseAllFileDispositionNodes(node *FileDispositionNode) {
	if node.HasChildren() {
		node.SetExpanded(false)
		for _, child := range node.Children() {
			collapseAllFileDispositionNodes(child)
		}
	}
}
