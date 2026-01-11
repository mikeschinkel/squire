package gomtui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/gomion/gommod/bubbletree"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
)

// SetDispositionCallback is called when a disposition changes in the UI
type SetDispositionCallback func(path dt.RelFilepath, disp FileDisposition) tea.Cmd

// FileDispositionModel manages the file disposition view UI and state.
// This view has: header, tree pane (left), content/table pane (right), footer.
// Implements tea.Model for BubbleTea integration.
type FileDispositionModel struct {
	// Layout dimensions
	terminalWidth  int
	terminalHeight int
	leftPaneWidth  int

	// UI components
	FolderTree      FileDispositionTreeModel // Hierarchical tree view of files
	FileContent     FileContentModel         // File content display (for file selection)
	FilesTable      FilesTableModel          // Files table display (for directory selection)
	IsDirectoryView bool                     // true = showing directory table, false = showing file content
	FocusPane       Pane                     // Which pane has focus (left or right)

	// Data sources
	fileSource      *FileSource            // Source of files to display
	UserRepo        *gitutils.Repo         // User repository
	ModuleDir       dt.DirPath             // Module directory path
	dispositionFunc DispositionFunc        // Callback to get disposition for a file
	setDisposition  SetDispositionCallback // Callback to notify parent of disposition changes
	FileCache       FileCache              // Cache of loaded file content
	GitStatusCache  gitutils.StatusMap     // Cache of git status
	RepoScoped      bool                   // true = full-repo, false = module-scoped

	// State
	Err     error           // Any error to display
	context context.Context // Context for async operations
}

// FileDispositionModelArgs contains parameters for creating a new FileDispositionModel
type FileDispositionModelArgs struct {
	FileSource      *FileSource
	UserRepo        *gitutils.Repo
	ModuleDir       dt.DirPath
	Width           int
	Height          int
	DispositionFunc DispositionFunc
	SetDisposition  SetDispositionCallback
	RepoScoped      bool
}

// NewFileDispositionModel creates a new file disposition model
func NewFileDispositionModel(args FileDispositionModelArgs) (m FileDispositionModel) {
	if args.Height == 0 || args.Width == 0 {
		panic("gomtui.NewFileDispositionModel called before height or width set via tea.WindowSizeMsg")
	}

	m = FileDispositionModel{
		terminalWidth:   args.Width,
		terminalHeight:  args.Height,
		fileSource:      args.FileSource,
		UserRepo:        args.UserRepo,
		ModuleDir:       args.ModuleDir,
		dispositionFunc: args.DispositionFunc,
		setDisposition:  args.SetDisposition,
		RepoScoped:      args.RepoScoped,
		FocusPane:       LeftPane,
		FileCache:       make(FileCache),
		context:         context.Background(),
	}

	// Create tree component
	if !args.FileSource.HasFiles() {
		m.FolderTree = NewEmptyFileDispositionTreeModel(
			"No changed files available for commit in current module.\n\nUse 'm' to toggle between module and repository scope.",
			m.PaneInnerHeight(),
			args.DispositionFunc,
		)
		goto end
	}

	m.FolderTree = NewFileDispositionTreeModel(args.FileSource, m.PaneInnerHeight(), args.DispositionFunc)
	m.FileContent = NewFileContentModel(m.RightPaneInnerWidth(), m.PaneInnerHeight())

end:
	return m
}

func (l FileDispositionModel) SelectedNodePath() (path dt.RelFilepath) {
	node := l.FolderTree.FocusedNode()
	if node == nil {
		goto end
	}
	path = node.Data().Path
end:
	return path
}

func (l FileDispositionModel) maybeChangeDisposition(msg tea.Msg) (cdm changeDispositionMsg) {
	path := l.SelectedNodePath()
	if path != "" {
		goto end
	}
	cdm = maybeChangeDisposition(path, msg)
end:
	return cdm
}

// Initialized returns true if terminal height and width are set, IOW if the
// constructor NewFileDispositionModel() as called to instantiate it.
func (l FileDispositionModel) Initialized() bool {
	return l.terminalWidth > 0 && l.terminalHeight > 0
}

// Resize updates the sizes of the layout'a components
func (l FileDispositionModel) Resize() FileDispositionModel {
	l.FolderTree = l.FolderTree.SetSize(l.LeftPaneWidth(), l.PaneInnerHeight())
	l.FileContent = l.FileContent.SetSize(l.RightPaneInnerWidth(), l.PaneInnerHeight())
	if l.IsDirectoryView {
		l.FilesTable = l.FilesTable.SetSize(l.RightPaneInnerWidth(), l.PaneHeight())
	}
	return l
}

func (l FileDispositionModel) NeedsResizing() bool {
	return l.FolderTree.HasTree() && l.leftPaneWidth == 0
}

// LeftPaneWidth returns the width for the left tree pane.
func (l FileDispositionModel) LeftPaneWidth() int {
	return l.FolderTree.LayoutWidth()
}

// RightPaneWidth returns the total width for the right pane including chrome.
// Used when rendering file content (which needs explicit width for pane wrapper).
//
// Calculation: terminalWidth - leftPaneWidth
// Note: leftPaneWidth is the content width; the actual rendered left pane
// adds its own chrome (borders + padding), but that's handled separately.
func (l FileDispositionModel) RightPaneWidth() int {
	return l.terminalWidth - l.LeftPaneWidth()
}

// RightPaneInnerWidth returns the width for the right content/table pane.
//
// IMPORTANT: The +2 offset is empirically determined due to lipgloss version mismatch:
// - Project uses lipgloss v1.1.0
// - bubble-table uses lipgloss v0.5.0
// - Border/padding calculations differ between versions
// - Theoretical calculation would be -6, but actual rendering requires +2
//
// When bubble-table is updated to lipgloss v2, this may need adjustment.
func (l FileDispositionModel) RightPaneInnerWidth() int {
	return l.terminalWidth - l.LeftPaneWidth() + 2
}

// PaneHeight returns the full pane height (outer, including borders).
// Used by: table (direct render), basePaneStyle wrapper
// Calculation: See layout_constants.go for chrome dimensions
func (l FileDispositionModel) PaneHeight() int {
	if l.IsDirectoryView {
		return l.terminalHeight - PaneHeightForDirectoryView
	}
	return l.terminalHeight - PaneHeightForFileView
}

// PaneInnerHeight returns the viewport height inside a pane (inner, excluding pane borders).
// Used by: tree viewport, file content viewport
// Calculation: See layout_constants.go for chrome dimensions
func (l FileDispositionModel) PaneInnerHeight() int {
	if l.IsDirectoryView {
		return l.PaneHeight() - PaneBorderLines
	}
	return l.PaneHeight()
}

// ============================================================================
// Helper methods
// ============================================================================

// togglePane switches focus between left and right panes
func (m FileDispositionModel) togglePane() FileDispositionModel {
	switch m.FocusPane {
	case LeftPane:
		m.FocusPane = RightPane
	default:
		m.FocusPane = LeftPane
	}
	return m
}

// loadFileContent loads file content, using cache if available
func (m FileDispositionModel) loadFileContent(path dt.RelFilepath) (content string, yOffset int, err error) {
	var filepath dt.Filepath
	var bytes []byte
	var actualPath dt.RelFilepath
	var cached *bubbletree.File
	var ok bool
	var pathStr string
	var parts []string

	// Check cache first
	cached, ok = m.FileCache[path]
	if ok {
		content = cached.Content
		yOffset = cached.YOffset
		goto end
	}

	// Handle renamed files (format: "oldpath -> newpath")
	// For renamed files, use the new path
	pathStr = string(path)
	if strings.Contains(pathStr, " -> ") {
		parts = strings.Split(pathStr, " -> ")
		actualPath = dt.RelFilepath(strings.TrimSpace(parts[1]))
	} else {
		actualPath = path
	}

	// Construct full path
	filepath = dt.FilepathJoin(m.UserRepo.Root, actualPath)

	// Read file
	bytes, err = filepath.ReadFile()
	if err != nil {
		err = NewErr(ErrGit, filepath.ErrKV(), err)
		goto end
	}

	content = string(bytes)

	// Cache it
	m.FileCache[path] = bubbletree.NewFile(path, content).SetData(NewFileData())
	yOffset = 0

end:
	return content, yOffset, err
}

// withUpdatedFileCache updates the cached scroll position for a file
func (m FileDispositionModel) withUpdatedFileCache(path dt.RelFilepath, yOffset int) FileDispositionModel {
	var updated bubbletree.File
	var newCache FileCache

	cached, ok := m.FileCache[path]
	if !ok {
		goto end
	}
	// Create new File with updated YOffset
	updated = *cached
	updated.YOffset = yOffset

	// Create new map with updated entry
	newCache = make(FileCache, len(m.FileCache))
	for k, v := range m.FileCache {
		newCache[k] = v
	}
	newCache[path] = &updated

	// Return new model with new cache
	m.FileCache = newCache
end:
	return m
}

// gitStatusMap returns the cached git status map, loading it if necessary.
func (m FileDispositionModel) gitStatusMap() (statusMap gitutils.StatusMap, err error) {
	// Return cached if available
	if m.GitStatusCache != nil {
		statusMap = m.GitStatusCache
		goto end
	}

	// Load git status
	statusMap, err = m.UserRepo.StatusMap(m.context, &gitutils.StatusArgs{
		HumanReadable: false,
	})

end:
	return statusMap, err
}

// setGitStatus enriches bubbletree.File with git status information
func (m FileDispositionModel) setGitStatus(f *bubbletree.File) {
	var status gitutils.FileStatus
	var found bool

	if !f.HasData() {
		f.SetData(NewFileData())
	}
	fileData, err := dtx.AssertType[*FileData](f.Data())
	if err != nil {
		panic(err.Error())
	}

	// Look up git status for this file
	status, found = m.GitStatusCache[f.Path]
	if !found {
		fileData.FileStatus = gitutils.FileStatus{}
		goto end
	}
	fileData.FileStatus = status
	f.SetData(fileData)
end:
	return
}

// handleLeftPaneFocus handles updates when left pane (tree) has focus
//
//goland:noinspection GoAssignmentToReceiver
func (m FileDispositionModel) handleLeftPaneFocus(msg tea.Msg) (FileDispositionModel, tea.Cmd) {
	var cmd tea.Cmd

	// Delegate navigation to tree
	m.FolderTree, cmd = m.FolderTree.Update(msg)

	// Update file content or directory table when tree selection changes
	selectedFile := m.FolderTree.SelectedFile()
	if selectedFile == nil {
		return m, cmd
	}

	selectedNode := m.FolderTree.FocusedNode()
	if selectedNode != nil && !selectedNode.HasChildren() {
		// File selected - show file content
		m.IsDirectoryView = false

		// Load with cache
		content, yOffset, err := m.loadFileContent(selectedFile.Path)
		if err != nil {
			content = fmt.Sprintf("Error loading file:\n%v", err)
			yOffset = 0
		}
		m.FileContent = m.FileContent.SetContent(content, yOffset)
		return m, cmd
	}

	// Directory selected - show directory table
	m.IsDirectoryView = true

	// Get child files from tree
	childFiles := GetNodeChildData[bubbletree.File](selectedNode)

	// Get or load git status
	var err error
	m.GitStatusCache, err = m.gitStatusMap()
	if err != nil {
		m.Err = err
		return m, cmd
	}

	// Batch load metadata for child files
	err = batchLoadMeta(childFiles, m.UserRepo.Root)
	if err != nil {
		m.Err = err
		return m, cmd
	}

	// Enrich with git status
	for _, file := range childFiles {
		m.GitStatusCache.EnsureFileStatus(file.Path)
		m.setGitStatus(file)
	}

	// Create directory object
	dir := Directory{
		Path:  dt.RelDirPath(selectedFile.Path),
		Files: childFiles,
	}

	// Create/update table
	m.FilesTable = NewFilesTableModel(dir,
		m.dispositionFunc,
		m.RightPaneInnerWidth(),
		m.PaneHeight(),
	)

	return m, cmd
}

// handleRightPaneFocus handles updates when right pane (content/table) has focus
//
//goland:noinspection GoAssignmentToReceiver
func (m FileDispositionModel) handleRightPaneFocus(msg tea.Msg) (FileDispositionModel, tea.Cmd) {
	var cmd tea.Cmd

	// Right pane has focus - delegate to either file content or directory table
	if m.IsDirectoryView {
		// Directory table has focus - handle navigation and disposition changes
		m.FilesTable, cmd = m.FilesTable.Update(msg)
	} else {
		// File content has focus - handle scrolling
		m.FileContent, cmd = m.FileContent.Update(msg)

		// Update cache with current scroll position
		selectedFile := m.FolderTree.SelectedFile()
		if selectedFile != nil {
			m = m.withUpdatedFileCache(selectedFile.Path, m.FileContent.YOffset())
		}
	}

	return m, cmd
}

// ============================================================================
// tea.Model interface implementation
// ============================================================================

// Ensure FileDispositionModel implements tea.Model
var _ tea.Model = (*FileDispositionModel)(nil)

// Init initializes the model (implements tea.Model)
func (m FileDispositionModel) Init() tea.Cmd {
	// Initialize tree component
	return m.FolderTree.Init()
}

// Update handles messages and updates the model (implements tea.Model)
//
//goland:noinspection GoAssignmentToReceiver
func (m FileDispositionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle different message types
	switch msg := msg.(type) {
	case resizeLayoutMsg:
		// Update dimensions and resize all components
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		m = m.Resize()

	case tea.KeyMsg:
		ms := msg.String()

		// Check for disposition change keys when tree has focus
		if m.FocusPane == LeftPane {
			result := m.maybeChangeDisposition(msg)
			if result.Disposition != UnspecifiedDisposition {
				// Apply disposition through callback
				if m.setDisposition != nil {
					cmd = m.setDisposition(result.RelFilepath, result.Disposition)
					cmds = append(cmds, cmd)
				}
			}
		}

		// Handle navigation keys
		switch ms {
		case "tab":
			m = m.togglePane()
		case "shift+tab":
			m = m.togglePane()
		}
	}

	// Delegate to focused pane
	switch m.FocusPane {
	case LeftPane:
		m, cmd = m.handleLeftPaneFocus(msg)
		cmds = append(cmds, cmd)
	case RightPane:
		m, cmd = m.handleRightPaneFocus(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the model (implements tea.Model)
func (m FileDispositionModel) View() string {
	// Check for errors
	if m.Err != nil {
		if errors.Is(m.Err, ErrNoChangedFiles) {
			return "No changed files to display.\n\nPress q to quit."
		}
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.Err)
	}

	var sb strings.Builder

	// Header
	scope := fmt.Sprintf("Module=%s", renderRGBColor(m.ModuleDir.ToTilde(dt.OrFullPath), GreenColor))
	if m.RepoScoped {
		scope = fmt.Sprintf("Repo=%s", m.UserRepo.Root)
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		Render("Commit Plan: " + scope)

	// Build menu
	menu := fmt.Sprintf("↑/↓:Navigate | ←/→:Expand/Collapse | %s:Commit | %s:Omit | %s:.gitignore | %s:.git/exclude | m:Module/Repo | Enter:Continue | q:Quit",
		CommitDisposition.Key(),
		OmitDisposition.Key(),
		GitIgnoreDisposition.Key(),
		GitExcludeDisposition.Key(),
	)
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color(SilverColor)).
		Render(menu)

	// Calculate border colors based on focus
	leftBorderColor := GrayColor
	rightBorderColor := GrayColor

	if m.FocusPane == LeftPane {
		leftBorderColor = CyanColor
	} else if m.FocusPane == RightPane {
		rightBorderColor = CyanColor
	}

	// Create styled panes
	basePaneStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		Height(m.PaneInnerHeight())

	leftPane := basePaneStyle.
		PaddingLeft(1).
		PaddingRight(1).
		BorderForeground(lipgloss.Color(leftBorderColor)).
		Render(m.FolderTree.View())

	// Render right pane based on view type
	var rightPane string
	if m.IsDirectoryView {
		w, h := m.RightPaneWidth(), m.PaneHeight()
		// Table already has its own borders - render directly
		m.FilesTable = m.FilesTable.SetBorderColor(rightBorderColor).SetSize(w, h)
		rightPane = m.FilesTable.View()
	} else {
		// File content needs a pane wrapper with borders
		rightPane = basePaneStyle.
			Width(m.RightPaneWidth()).
			PaddingLeft(1).
			Height(m.PaneHeight()).
			BorderForeground(lipgloss.Color(rightBorderColor)).
			Render(m.FileContent.View())
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	n := strings.Count(body, "\n")
	if n >= 0 {
		print("")
	}

	sb.WriteString(header)
	sb.WriteString("\n")
	sb.WriteString(body)
	sb.WriteString("\n")
	sb.WriteString(footer)

	return sb.String()
}
