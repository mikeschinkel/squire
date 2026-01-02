package gomtui

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/mikeschinkel/gomion/gommod/bubbletree"
)

// FileDispositionNodeProvider implements bubbletree.NodeProvider for File nodes
type FileDispositionNodeProvider struct {
	bubbletree.NodeProvider[File]
}

// NewFileDispositionNodeProvider creates a new file node provider with compact 2-space indentation
func NewFileDispositionNodeProvider() *FileDispositionNodeProvider {
	return &FileDispositionNodeProvider{
		NodeProvider: bubbletree.NewCompactNodeProvider[File](bubbletree.TriangleExpanderControls),
		//NodeProvider: bubbletree.NewCompactNodeProvider[File](bubbletree.PlusExpanderControls),
		//NodeProvider: bubbletree.NewCompactNodeProvider[File](bubbletree.NoExpanderControls),
	}
}

// Text returns the formatted display name (filename)
func (p *FileDispositionNodeProvider) Text(node *FileDispositionNode) string {
	return filepath.Base(string(node.Data().Path))
}

// Suffix returns the file disposition suffix
func (p *FileDispositionNodeProvider) Suffix(node *FileDispositionNode) string {
	d := node.Data().Disposition
	return fmt.Sprintf("[%s]", renderRGBColor(d.Key(), d.RGBColor()))
}

//// Icon returns the expand/collapse indicator for folders, empty for files
//func (p *FileDispositionNodeProvider) Icon(node *FileDispositionNode) string {
//	if node.HasChildren() {
//		if node.IsExpanded() {
//			return "" // Expanded folder - no icon, tree structure shows it
//		}
//		return "+" // Collapsed folder - show plus to indicate can expand
//	}
//	return "" // Files don't need an icon - branch character is enough
//}

// Style returns the lipgloss style based on disposition and focus state
func (p *FileDispositionNodeProvider) Style(node *FileDispositionNode, tree *bubbletree.Tree[File]) (style lipgloss.Style) {
	style = styleWithRGBColor(node.Data().Disposition.RGBColor())
	if tree.IsFocusedNode(node) {
		// Use inverse video for better accessibility
		return style.Reverse(true)
	}

	// Use disposition color for non-focused items
	return style
}

//// BuildPrefix builds the tree structure prefix with compact indentation
//func (p *FileDispositionNodeProvider) BuildPrefix(node *FileDispositionNode, ancestorIsLastChild []bool, isLast bool) string {
//	// DEBUG: Print what we're receiving
//	// fmt.Printf("BuildPrefix for %s: ancestors=%v, isLast=%v, depth=%d\n",
//	//	node.Name(), ancestorIsLastChild, isLast, node.Depth())
//
//	return p.NodeProvider.BuildPrefix(node, ancestorIsLastChild, isLast)
//}
