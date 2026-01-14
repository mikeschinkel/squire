package gomtui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/gomion/gommod/bubbletree"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
	"github.com/mikeschinkel/gomion/gommod/gompkg"
)

// File size constants for content loading
const (
	MaxFileContentSize = 32 * 1024  // 32KB - full load without warning
	MaxFilePreviewSize = 128 * 1024 // 128KB - max preview for large files
)

// SetDispositionCallback is called when a disposition changes in the UI
type SetDispositionCallback func(path dt.RelFilepath, disp gompkg.FileDisposition) tea.Cmd

// FileDispositionModel manages the file disposition view UI and state.
// This view has: header, tree pane (left), content/table pane (right), footer.
// Implements tea.Model for BubbleTea integration.
type FileDispositionModel struct {
	Logger *slog.Logger

	// Layout dimensions
	terminalWidth  int
	terminalHeight int
	leftPaneWidth  int

	// UI components
	FolderTree    FileDispositionTreeModel // Hierarchical tree view of files
	FileContent   FileContentModel         // File content display (for file selection)
	FilesTable    FilesTableModel          // Files table display (for directory selection)
	IsContentView bool                     // false = showing directory table, true = showing file content
	FocusPane     Pane                     // Which pane has focus (left or right)

	dirMetaLoaded map[dt.RelDirPath]struct{} // Track when a directory has had its file meta loaded

	// Data sources
	commitPlan  *gompkg.CommitPlan // Reference to commit plan
	fileSource  *FileSource        // Source of files to display
	UserRepo    *gitutils.Repo     // User repository
	moduleDir   dt.DirPath         // Module directory path
	modulePath  dt.RelDirPath      // Relative path for module from repoRoot; "" if they are the same
	FileCache   FileCache          // Cache of loaded file content
	RepoScoped  bool               // true = full-repoRoot, false = module-scoped
	CommitScope gompkg.CommitScope // "module" or "repoRoot"

	// Commit plan persistence (auto-save)
	saveSeq      int           // Auto-save sequence number (incremented on each disposition change)
	saveDebounce time.Duration // Debounce period (default 3 seconds)

	// Selection tracking (to detect changes)
	lastSelectedNode *bubbletree.FileNode // ← ADD THIS

	// State
	Err     error           // Any error to display
	context context.Context // Context for async operations
}

// FileDispositionModelArgs contains parameters for creating a new FileDispositionModel
type FileDispositionModelArgs struct {
	Logger     *slog.Logger
	UserRepo   *gitutils.Repo
	ModulePath dt.RelDirPath
	CommitPlan *gompkg.CommitPlan
	FileSource *FileSource
	Width      int
	Height     int
	RepoScoped bool
}

// NewFileDispositionModel creates a new file disposition model
func NewFileDispositionModel(args FileDispositionModelArgs) (m FileDispositionModel) {
	if args.Height == 0 || args.Width == 0 {
		panic("gomtui.NewFileDispositionModel called before height or width set via tea.WindowSizeMsg")
	}

	m = FileDispositionModel{
		Logger:         args.Logger,
		fileSource:     args.FileSource,
		terminalWidth:  args.Width,
		terminalHeight: args.Height,
		UserRepo:       args.UserRepo,
		modulePath:     args.ModulePath,
		commitPlan:     args.CommitPlan,
		saveDebounce:   3 * time.Second,
		moduleDir:      dt.DirPathJoin(args.UserRepo.Root, args.ModulePath),
		RepoScoped:     args.RepoScoped,
		FocusPane:      LeftPane,
		FileCache:      make(FileCache),
		dirMetaLoaded:  make(map[dt.RelDirPath]struct{}),
		context:        context.Background(),
	}

	// Create tree component
	if !args.FileSource.HasFiles() {
		m.FolderTree = NewEmptyFileDispositionTreeModel(
			"No changed files available for commit in current module.\n\nUse 'm' to toggle between module and repository scope.",
			m.PaneInnerHeight(),
			m.DispositionFunc(),
		)
		goto end
	}

	m.FolderTree = NewFileDispositionTreeModel(FileDispositionTreeModelArgs{
		FileSource:      args.FileSource,
		Height:          m.PaneInnerHeight(),
		DispositionFunc: m.DispositionFunc(),
		Logger:          args.Logger,
	})
	m.FileContent = NewFileContentModel(
		m.RightPaneInnerWidth(),
		m.PaneInnerHeight(),
		args.Logger,
	)

end:
	return m
}

// Ensure FileDispositionModel implements tea.Model
var _ tea.Model = (*FileDispositionModel)(nil)

// Init initializes the model (implements tea.Model)
func (m FileDispositionModel) Init() tea.Cmd {
	m.Logger.Info("FileDispositionModel.Init()")
	// Initialize tree component
	treeCmd := m.FolderTree.Init()

	// Trigger initial directory load for the focused node
	initialNode := m.FolderTree.FocusedNode()
	if initialNode.HasChildren() {
		m.IsContentView = false
		m.lastSelectedNode = initialNode
		return tea.Batch(treeCmd, m.requestDirectoryViewCmd(initialNode))
	}

	return treeCmd
}

// requestDirectoryViewCmd handles loading or reloading a directory view
// Returns a command to load/reload the directory table
func (m FileDispositionModel) requestDirectoryViewCmd(node *bubbletree.FileNode) tea.Cmd {
	if !node.HasChildren() {
		return nil
	}

	file := m.FolderTree.SelectedFile()
	if file == nil {
		return nil
	}

	relPath := dt.RelDirPath(file.Path)
	childFiles := GetNodeChildData[bubbletree.File](node)

	// Check if already loaded
	if _, loaded := m.dirMetaLoaded[relPath]; loaded {
		m.Logger.Info("FileDispositionModel.requestDirectoryViewCmd()", "path", relPath, "cached", true)
		// Already loaded - just reload the table (no I/O)
		return requestReloadTableCmd(relPath, childFiles)
	}

	m.Logger.Info("FileDispositionModel.requestDirectoryViewCmd()", "path", relPath, "cached", false, "child_count", len(childFiles))
	// Not loaded yet - trigger directory view load (will do I/O)
	return requestDirectoryViewCmd(relPath, childFiles)
}

// Update handles messages and updates the model (implements tea.Model)
//
//goland:noinspection GoAssignmentToReceiver
func (m FileDispositionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	m.Logger.Info("FileDispositionModel.Update()", teaMsgAttrs(msg))

	switch msg := msg.(type) {
	case tea.KeyMsg:

		switch msg.String() {
		case "m":
			m = m.handleModuleToggle()
			// Call Init() again
			cmds = appendCmd(cmds, m.Init())

		case "tab", "shift+tab":
			// Handle pane navigation keys
			m = m.togglePane()

		default:
			if m.FocusPane == LeftPane {
				// See if key pressed for KeyMsg indicates a valid file disposition to change
				// when tree has focus.
				cmds = appendCmd(cmds, m.requestDispositionChangeCmd(msg))
			}

			// TODO: Should we delegate to Foldertree.Update() or other kids here?
		}

	case screenDimensionsMsg:
		// Capture the screen dimensions
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height

	case setCommitPlanMsg:
		// Get a local copy of the commit plan
		m.commitPlan = msg.plan
		// TODO: ???
		cmds = appendCmd(cmds, nil)

	case requestFileDispositionUIMsg:
		// TODO: ???
		cmds = appendCmd(cmds, nil)

	case refreshTreeMsg:
		// Placeholder for refershing the tree

	case refreshTableMsg:
		// Placeholder for refershing the table

	case refreshDispositionLayoutMsg:
		cmds = appendCmd(cmds, tea.Batch(
			refreshTreeCmd(),
			refreshTableCmd(),
		))

	case changeDispositionMsg:
		// Handle disposition changes from table/tree view
		// If node is provided (directory in tree view), cascade to all descendants
		// Otherwise just update the single path (file in tree view or table view)
		m = m.changeDisposition(msg)
		// Batch both save and table refresh for async pattern
		cmds = appendCmd(cmds, tea.Batch(
			scheduleCommitPlanSaveCmd(),
			refreshDispositionLayoutCmd(),
		))

	case resizeLayoutMsg:
		// Update dimensions and resize all components
		m = m.Resize()

	// ========================================================================
	// Async File Loading Handlers (handleLeftPaneFocus refactoring)
	// ========================================================================

	case loadFileContentMsg:
		// Check cache first (fast, non-blocking)
		if _, ok := m.FileCache[msg.path]; ok {
			// Already cached - just display it
			cachedFile := m.FileCache[msg.path]
			m.FileContent.SetContent(cachedFile.Content, cachedFile.YOffset)
			break
		}
		// Cache miss - trigger async load (I/O happens in background)
		cmd = loadFileContentCmd(m.UserRepo.Root, msg.path)
		cmds = appendCmd(cmds, cmd)

	case fileContentLoadedMsg:
		// File content loaded successfully - cache it and display
		m.FileCache[msg.path] = bubbletree.NewFile(msg.path, msg.content)
		m.FileContent = m.FileContent.SetContent(msg.content, msg.yOffset)

	case reloadTableMsg:
		dir := Directory{
			RelPath: msg.relDirPath,
			Files:   msg.childFiles,
		}
		m.FilesTable = NewFilesTableModel(dir,
			m.DispositionFunc(),
			m.RightPaneInnerWidth(),
			m.PaneHeight(),
		)

		m.dirMetaLoaded[dir.RelPath] = struct{}{}

		// TODO: Re-enable background loading after fixing Command pattern
		// After table created, identify small files and start sequential loading
		//var smallFiles []dt.RelFilepath
		//for _, file := range msg.childFiles {
		//	if file.Meta() != nil && file.Meta().Size < MaxFileContentSize {
		//		smallFiles = append(smallFiles, file.RelPath)
		//	}
		//}
		//
		//if len(smallFiles) > 0 {
		//	cmd = loadNextFileContentCmd(smallFiles[0], smallFiles[1:])
		//	cmds = appendCmd(cmds, cmd)
		//}

	// ========================================================================
	// Background Loading Handlers (Sequential, Event-Driven)
	// ========================================================================

	case loadFileMetaMsg:
		// Check if already loaded (user may have requested it manually)
		if !msg.file.HasMeta() {
			// Load metadata for this one file
			err := msg.file.LoadMeta(m.UserRepo.Root)
			if err != nil {
				// Log error but continue with next file (don't break the chain)
				// Silently ignore errors in background loading
			}
		}

		// Chain to next file in queue (with throttle delay)
		if len(msg.remainingFiles) > 0 {
			cmd = tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
				return loadFileMetaMsg{
					file:           msg.remainingFiles[0],
					remainingFiles: msg.remainingFiles[1:],
				}
			})
			cmds = appendCmd(cmds, cmd)
		}

	case loadNextFileContentMsg:
		// Check if already cached (user may have requested it manually)
		if _, ok := m.FileCache[msg.path]; !ok {
			// Load file content
			content, yOffset, err := loadFileContent(m.UserRepo.Root, msg.path)
			if err == nil {
				// Cache it (no need to show in UI, just background caching)
				// Cache update happens synchronously in Update(), no race conditions
				m.FileCache[msg.path] = bubbletree.NewFile(msg.path, content).SetData(NewFileData())
				m.FileCache[msg.path].YOffset = yOffset
			}
		}

		// Chain to next file in queue (with throttle delay)
		if len(msg.remainingPaths) > 0 {
			cmd = tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
				return loadNextFileContentMsg{
					path:           msg.remainingPaths[0],
					remainingPaths: msg.remainingPaths[1:],
				}
			})
			cmds = appendCmd(cmds, cmd)
		}

	}
	if m.Ready() {

		// Delegate to focused pane
		switch m.FocusPane {
		case LeftPane:
			m, cmd = m.handleLeftPaneFocus(msg)
			cmds = appendCmd(cmds, cmd)
		case RightPane:
			m, cmd = m.handleRightPaneFocus(msg)
			cmds = appendCmd(cmds, cmd)
		default:
			// Here to stop Goland from complaining
		}
	}
	return m, tea.Batch(cmds...)
}

var (
	ErrLoadingFile = errors.New("error loading file")
)

//goland:noinspection GoAssignmentToReceiver
func (m FileDispositionModel) changeDisposition(msg changeDispositionMsg) FileDispositionModel {
	var nodeType string
	switch {
	case msg.Node != nil:
		// Directory — cascade to all descendants
		for _, path := range GetAllDescendantPaths(msg.Node) {
			m = m.SetDisposition(path, msg.Disposition)
		}
		nodeType = "directory"
	default:
		// File — just update the single path
		m = m.SetDisposition(msg.Path, msg.Disposition)
		nodeType = "file"
	}
	m.Logger.Info("FileDispositionModel.changeDisposition()", teaMsgAttrs(msg), "node_type", nodeType)
	return m
}

// View renders the model (implements tea.Model)
func (m FileDispositionModel) View() string {
	m.Logger.Info("FileDispositionModel.View()")
	// Check for errors
	if m.Err != nil {
		if errors.Is(m.Err, ErrNoChangedFiles) {
			return "No changed files to display.\n\nPress q to quit."
		}
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.Err)
	}

	var sb strings.Builder

	// Header
	scope := fmt.Sprintf("Module=%s", renderRGBColor(m.moduleDir.ToTilde(dt.OrFullPath), GreenColor))
	if m.RepoScoped {
		scope = fmt.Sprintf("Repo=%s", m.UserRepo.Root)
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		Render("Commit Plan: " + scope)

	// Build menu
	menu := fmt.Sprintf("↑/↓:Navigate | ←/→:Expand/Collapse | %s:Commit | %s:Omit | %s:.gitignore | %s:.git/exclude | m:Module/Repo | Enter:Continue | q:Quit",
		gompkg.CommitDisposition.Key(),
		gompkg.OmitDisposition.Key(),
		gompkg.GitIgnoreDisposition.Key(),
		gompkg.GitExcludeDisposition.Key(),
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
	switch {
	case m.IsContentView:
		// File content needs a pane wrapper with borders
		rightPane = basePaneStyle.
			Width(m.RightPaneWidth()).
			PaddingLeft(1).
			Height(m.PaneHeight()).
			BorderForeground(lipgloss.Color(rightBorderColor)).
			Render(m.FileContent.View())
	default:
		w, h := m.RightPaneWidth(), m.PaneHeight()
		// Table already has its own borders - render directly
		m.FilesTable = m.FilesTable.SetBorderColor(rightBorderColor).SetSize(w, h)
		rightPane = m.FilesTable.View()
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	sb.WriteString(header)
	sb.WriteString("\n")
	sb.WriteString(body)
	sb.WriteString("\n")
	sb.WriteString(footer)

	return sb.String()
}

// Ready  returns true if this model is ready to accept update messages
func (m FileDispositionModel) Ready() bool {
	return m.Logger != nil &&
		m.HasDimensions() &&
		m.fileSource != nil &&
		m.FolderTree.HasTree()
}

// HasDimensions returns true if terminal height and width are set, IOW if the
// constructor NewFileDispositionModel() as called to instantiate it.
func (m FileDispositionModel) HasDimensions() bool {
	return m.terminalWidth > 0 && m.terminalHeight > 0
}

// Resize updates the sizes of the layout'a components
func (m FileDispositionModel) Resize() FileDispositionModel {
	m.Logger.Info("FileDispositionModel.Resize()")
	m.FolderTree = m.FolderTree.SetSize(m.LeftPaneWidth(), m.PaneInnerHeight())
	m.FileContent = m.FileContent.SetSize(m.RightPaneInnerWidth(), m.PaneInnerHeight())
	if !m.IsContentView {
		m.FilesTable = m.FilesTable.SetSize(m.RightPaneInnerWidth(), m.PaneHeight())
	}
	return m
}

// LeftPaneWidth returns the width for the left tree pane.
func (m FileDispositionModel) LeftPaneWidth() int {
	return m.FolderTree.LayoutWidth()
}

// RightPaneWidth returns the total width for the right pane including chrome.
// Used when rendering file content (which needs explicit width for pane wrapper).
//
// Calculation: terminalWidth - leftPaneWidth
// Note: leftPaneWidth is the content width; the actual rendered left pane
// adds its own chrome (borders + padding), but that's handled separately.
func (m FileDispositionModel) RightPaneWidth() int {
	return m.terminalWidth - m.LeftPaneWidth()
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
func (m FileDispositionModel) RightPaneInnerWidth() int {
	return m.terminalWidth - m.LeftPaneWidth() + 2
}

// PaneHeight returns the full pane height (outer, including borders).
// Used by: table (direct render), basePaneStyle wrapper
// Calculation: See layout_constants.go for chrome dimensions
func (m FileDispositionModel) PaneHeight() int {
	if m.IsContentView {
		return m.terminalHeight - PaneHeightForFileView
	}
	return m.terminalHeight - PaneHeightForDirectoryView
}

// PaneInnerHeight returns the viewport height inside a pane (inner, excluding pane borders).
// Used by: tree viewport, file content viewport
// Calculation: See layout_constants.go for chrome dimensions
func (m FileDispositionModel) PaneInnerHeight() int {
	if m.IsContentView {
		return m.PaneHeight()
	}
	return m.PaneHeight() - PaneBorderLines
}

// Disposition returns the disposition for a file path.
// Returns UnspecifiedDisposition if no explicit disposition has been set.
func (m FileDispositionModel) Disposition(path dt.RelFilepath) (disp gompkg.FileDisposition) {
	return m.commitPlan.GetFileDisposition(path)
}

// SetDisposition updates the disposition for a file path (mutates map in-place).
// The map is accessed via pointer, so changes are visible to all callbacks.
// For recursive updates (directories), handle at call site.
func (m FileDispositionModel) SetDisposition(path dt.RelFilepath, disp gompkg.FileDisposition) FileDispositionModel {
	m.Logger.Info("FileDispositionModel.SetDisposition()", "path", path, "new_disposition", disp.String())
	// Update both the dispositions map and the cache
	m.commitPlan.SetFileDisposition(path, disp)
	return m
}

func (m FileDispositionModel) Context() context.Context {
	if m.context == nil {
		m.context = context.Background()
	}
	return m.context
}

type DispositionFunc func(path dt.RelFilepath) gompkg.FileDisposition

// DispositionFunc returns a func that provides the disposition for a file path.
func (m FileDispositionModel) DispositionFunc() DispositionFunc {
	return m.Disposition
}

// togglePane switches focus between left and right panes
func (m FileDispositionModel) togglePane() FileDispositionModel {
	m.Logger.Info("FileDispositionModel.togglePane()", "old_pane", m.FocusPane.String())
	switch m.FocusPane {
	case LeftPane:
		m.FocusPane = RightPane
	default:
		m.FocusPane = LeftPane
	}
	m.Logger.Info("FileDispositionModel.togglePane()", "new_pane", m.FocusPane.String())
	return m
}

// withUpdatedFileCache updates the cached scroll position for a file
func (m FileDispositionModel) withUpdatedFileCache(path dt.RelFilepath, yOffset int) FileDispositionModel {
	var updated bubbletree.File
	var newCache FileCache

	m.Logger.Info("FileDispositionModel.withUpdatedFileCache()", "files_cached", len(m.FileCache))

	cached, ok := m.FileCache[path]
	if !ok {
		m.Logger.Info("FileDispositionModel.withUpdatedFileCache()", "filepath", cached.Path, "status", "not_found")
		goto end
	}
	m.Logger.Info("FileDispositionModel.withUpdatedFileCache()", "filepath", cached.Path, "status", "found")

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

// handleLeftPaneFocus handles updates when left pane (tree) has focus
// Refactored to use async message-driven loading (no blocking I/O in hot path)
//
//goland:noinspection GoAssignmentToReceiver
func (m FileDispositionModel) handleLeftPaneFocus(msg tea.Msg) (_ FileDispositionModel, cmd tea.Cmd) {
	m.Logger.Info("FileDispositionModel.handleLeftPaneFocus()", teaMsgAttrs(msg))

	// Remember previous selection BEFORE delegating to tree
	previousNode := m.FolderTree.FocusedNode()

	m.Logger.Info("bubbletree.Update()", teaMsgAttrs(msg))
	// Delegate message to tree (may change selection)
	m.FolderTree, cmd = m.FolderTree.Update(msg)

	// Get current selection AFTER tree processing
	currentNode := m.FolderTree.FocusedNode()

	// Only proceed if selection ACTUALLY CHANGED
	// This prevents the message feedback loop: loadDirectoryViewMsg won't trigger another loadDirectoryViewMsg
	if currentNode == previousNode {
		m.Logger.Info("FileDispositionModel.handleLeftPaneFocus()", "node_status", "unchanged")
		return m, cmd // Selection unchanged, no need to check/load
	}
	m.Logger.Info("FileDispositionModel.handleLeftPaneFocus()", "node_status", "changed")

	// Selection changed - update tracking
	m.lastSelectedNode = currentNode

	// Check what's now selected and trigger loads (no I/O here)
	selectedFile := m.FolderTree.SelectedFile()
	if selectedFile == nil {
		m.Logger.Info("FileDispositionModel.handleLeftPaneFocus()", "node_status", "nil")
		return m, cmd
	}

	selectedNode := currentNode
	if !selectedNode.HasChildren() {
		m.Logger.Info("FileDispositionModel.handleLeftPaneFocus()", "node_type", "file")
		// File selected - load file content (non-blocking)
		m.IsContentView = true

		// Only emit load command if not already cached
		if _, cached := m.FileCache[selectedFile.Path]; !cached {
			return m, tea.Batch(cmd, requestFileContentCmd(selectedFile.Path))
		}

		// Already cached - just display it
		if cachedFile, ok := m.FileCache[selectedFile.Path]; ok {
			m.FileContent = m.FileContent.SetContent(cachedFile.Content, cachedFile.YOffset)
		}
		goto end
	}
	m.Logger.Info("FileDispositionModel.handleLeftPaneFocus()", "node_type", "directory")

	// Directory selected - load/reload directory view
	m.IsContentView = false
	cmd = tea.Batch(cmd, m.requestDirectoryViewCmd(selectedNode))

end:
	return m, cmd
}

// handleRightPaneFocus handles updates when right pane (content/table) has focus
//
//goland:noinspection GoAssignmentToReceiver
func (m FileDispositionModel) handleRightPaneFocus(msg tea.Msg) (FileDispositionModel, tea.Cmd) {
	var cmd tea.Cmd

	m.Logger.Info("FileDispositionModel.handleRightPaneFocus()", teaMsgAttrs(msg))

	// Right pane has focus - delegate to either file content or directory table
	switch {
	case m.IsContentView:
		m.Logger.Info("FileDispositionModel.handleLeftPaneFocus()", "node_type", "file")
		// File content has focus - handle scrolling
		m.FileContent, cmd = m.FileContent.Update(msg)

		// Update cache with current scroll position
		selectedFile := m.FolderTree.SelectedFile()
		if selectedFile != nil {
			m.Logger.Info("FileDispositionModel.handleLeftPaneFocus()", "selected_file", "found")
			m = m.withUpdatedFileCache(selectedFile.Path, m.FileContent.YOffset())
		}
	default:
		m.Logger.Info("FileDispositionModel.handleLeftPaneFocus()", "node_type", "directory")
		// Directory table has focus - handle navigation and disposition changes
		m.FilesTable, cmd = m.FilesTable.Update(msg)
	}

	return m, cmd
}

// handleModuleToggle switches between module-scoped and full-repoRoot view
func (m FileDispositionModel) handleModuleToggle() FileDispositionModel {
	switch m.CommitScope {
	case gompkg.RepoScope:
		m.CommitScope = gompkg.ModuleScope
	case gompkg.ModuleScope:
		m.CommitScope = gompkg.RepoScope
	default:
		dtx.Panicf("Invalid commit scope: %s", m.CommitScope)
	}
	m.Logger.Info("FileDispositionModel.handleModuleToggle()", "new_scope", m.CommitScope)
	return m
}

// maybeChangeDisposition inspects a key message to so it the kep-press matches
// the key associated with a FileDisposition, and if yet it changes the
// disposition for that file or all descendent files in a directory.
func (m FileDispositionModel) requestDispositionChangeCmd(msg tea.Msg) (cmd tea.Cmd) {
	var fd gompkg.FileDisposition
	var node *bubbletree.Node[bubbletree.File]

	m.Logger.Info("FileDispositionModel.requestDispositionChangeCmd()", teaMsgAttrs(msg))

	// See if key pressed for KeyMsg indicates a valid file disposition to change.
	fd = extractDispositionFromKeyMsg(msg)
	if !fd.IsValid() {
		m.Logger.Info("FileDispositionModel.requestDispositionChangeCmd()", "invalid_disposition", fd.String())
		goto end
	}

	node = m.FolderTree.FocusedNode()
	if node == nil {
		m.Logger.Info("FileDispositionModel.requestDispositionChangeCmd()", "focused_node", "nil")
		// No node selected, nothing to do
		goto end
	}

	// If this is a directory (has children), pass the node for cascading
	// If this is a file, just pass the path
	m.Logger.Info("FileDispositionModel.requestDispositionChangeCmd()", "has_children", node.HasChildren())
	switch {
	case node.HasChildren():
		cmd = requestDispositionChangeCmd(node, fd)
	default:
		cmd = requestDispositionChangeCmd(node.Data().Path, fd)
	}
end:
	return cmd
}

func extractDispositionFromKeyMsg(msg tea.Msg) (fd gompkg.FileDisposition) {
	// Handle disposition change keys (c/o/g/e)
	keyMsg := keyMsgString(msg)
	if len(keyMsg) != 1 {
		goto end
	}
	fd = gompkg.FileDisposition(keyMsg[0])
end:
	return fd
}

func keyMsgString(msg tea.Msg) (key string) {
	// Handle disposition change keys (c/o/g/e)
	keyMsg, err := dtx.AssertType[tea.KeyMsg](msg)
	if err != nil {
		goto end
	}
	key = keyMsg.String()
end:
	return key
}
