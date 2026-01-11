package gomtui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/bubbletree"
)

// FileDispositionNodeProvider implements bubbletree.NodeProvider for File nodes
type FileDispositionNodeProvider struct {
	bubbletree.NodeProvider[bubbletree.File]
	dispositionFn func(dt.RelFilepath) FileDisposition
}

// NewFileDispositionNodeProvider creates a new file node provider with compact 2-space indentation
func NewFileDispositionNodeProvider(dispositionFn func(dt.RelFilepath) FileDisposition) *FileDispositionNodeProvider {
	controls := bubbletree.TriangleExpanderControls
	//controls:= bubbletree.PlusExpanderControls
	//controls:= bubbletree.NoExpanderControls
	return &FileDispositionNodeProvider{
		NodeProvider:  bubbletree.NewCompactNodeProvider[bubbletree.File](controls),
		dispositionFn: dispositionFn,
	}
}

// Text returns the formatted display name (filename)
func (p *FileDispositionNodeProvider) Text(node *FileDispositionNode) string {
	return string(node.Data().Path.Base())
}

// Suffix returns the file disposition suffix
func (p *FileDispositionNodeProvider) Suffix(node *FileDispositionNode) string {
	path := node.Data().Path
	d := p.dispositionFn(path)
	return fmt.Sprintf("[%s]", renderRGBColor(d.Key(), d.RGBColor()))
}

// Style returns the lipgloss style based on disposition and focus state
func (p *FileDispositionNodeProvider) Style(node *bubbletree.FileNode, tree *bubbletree.Tree[bubbletree.File]) (style lipgloss.Style) {
	path := node.Data().Path
	d := p.dispositionFn(path)
	style = styleWithRGBColor(d.RGBColor())
	if tree.IsFocusedNode(node) {
		// Use inverse video for better accessibility
		return style.Reverse(true)
	}

	// Use disposition color for non-focused items
	return style
}
