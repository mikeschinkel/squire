package gomtui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// FileContentModel wraps viewport for displaying file content
type FileContentModel struct {
	viewport viewport.Model
	content  string
	width    int
	height   int
}

// NewFileContentModel creates a new file content model
func NewFileContentModel(width, height int) FileContentModel {
	vp := viewport.New(width, height)
	return FileContentModel{
		viewport: vp,
		width:    width,
		height:   height,
	}
}

// Init initializes the model
func (m FileContentModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m FileContentModel) Update(msg tea.Msg) (FileContentModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the content
func (m FileContentModel) View() string {
	return m.viewport.View()
}

// SetContent updates the displayed content and sets scroll position
func (m FileContentModel) SetContent(content string, yOffset int) FileContentModel {
	m.content = content
	m.viewport.SetContent(content)
	m.viewport.YOffset = yOffset
	return m
}

// YOffset returns the current scroll position
func (m FileContentModel) YOffset() int {
	return m.viewport.YOffset
}

// SetYOffset sets the scroll position
func (m FileContentModel) SetYOffset(yOffset int) FileContentModel {
	m.viewport.YOffset = yOffset
	return m
}

// SetSize updates the model dimensions
func (m FileContentModel) SetSize(width, height int) FileContentModel {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
	return m
}
