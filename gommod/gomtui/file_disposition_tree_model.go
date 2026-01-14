package gomtui

import (
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/bubbletree"
	"github.com/mikeschinkel/gomion/gommod/gompkg"
)

// FileDispositionTreeModel wraps bubbletree.Model for hierarchical file display
type FileDispositionTreeModel struct {
	Logger *slog.Logger
	model  bubbletree.Model[bubbletree.File]
}
type FileDispositionTreeModelArgs struct {
	FileSource      *FileSource
	Height          int
	DispositionFunc func(dt.RelFilepath) gompkg.FileDisposition
	Logger          *slog.Logger
}

// NewFileDispositionTreeModel creates a new folder tree model from a flat list of files
func NewFileDispositionTreeModel(args FileDispositionTreeModelArgs) (m FileDispositionTreeModel) {

	tree := NewFileDispositionTree(args.FileSource, args.DispositionFunc)

	// Create BubbleTea model
	m.model = bubbletree.NewModel(tree.tree, args.Height)
	m.Logger = args.Logger

	return m
}

// NewEmptyFileDispositionTreeModel creates an empty tree with a message
func NewEmptyFileDispositionTreeModel(message string, height int, dispositionFn func(dt.RelFilepath) gompkg.FileDisposition) FileDispositionTreeModel {
	// Create a single node with the message
	nodes := []*bubbletree.FileNode{
		bubbletree.NewNode("empty", message, bubbletree.File{
			Path: dt.RelFilepath(""),
		}),
	}

	tree := bubbletree.NewTree[bubbletree.File](nodes, &bubbletree.TreeArgs[bubbletree.File]{
		NodeProvider: NewFileDispositionNodeProvider(dispositionFn),
	})

	return FileDispositionTreeModel{
		model: bubbletree.NewModel(tree, height),
	}
}

func (m FileDispositionTreeModel) HasTree() bool {
	return m.model.Tree() != nil
}

// Init initializes the model
func (m FileDispositionTreeModel) Init() tea.Cmd {
	return m.model.Init()
}

// Update handles messages and updates the model
func (m FileDispositionTreeModel) Update(msg tea.Msg) (_ FileDispositionTreeModel, cmd tea.Cmd) {
	// Handle refreshTableMsg to trigger tree re-render (for disposition changes)
	m.Logger.Info("FileDispositionTreeModel.Update()", teaMsgAttrs(msg))
	switch msg := msg.(type) {
	case refreshTableMsg:
		// Tree nodes get their styling from the node provider which queries dispositions
		// Just re-rendering will pick up the new disposition colors
		// Force a refresh by calling the model's update with nil to trigger re-render
	default:
		m.Logger.Info("bubbletree.Model.Update()", teaMsgAttrs(msg))
		m.model, cmd = m.model.Update(msg)
	}
	return m, cmd
}

// View renders the tree
func (m FileDispositionTreeModel) View() string {
	return m.model.View()
}

// SelectedFile returns the currently selected file/folder
func (m FileDispositionTreeModel) SelectedFile() *bubbletree.File {
	node := m.model.FocusedNode()
	if node == nil {
		return nil
	}
	return node.Data()
}

// FocusedNode returns the currently selected node
func (m FileDispositionTreeModel) FocusedNode() *bubbletree.FileNode {
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
