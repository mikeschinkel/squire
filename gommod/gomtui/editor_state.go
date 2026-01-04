package gomtui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
	"github.com/mikeschinkel/gomion/gommod/gomcfg"
	"github.com/mikeschinkel/gomion/gommod/gompkg"
)

// EditorState is the main bubbletea model for the GRU staging editor
type EditorState struct {
	// Repository state
	CachedRepo    *gitutils.CachedWorktree
	UserRepo      *gitutils.Repo
	ModuleDir     dt.DirPath
	ModuleRelPath dt.RelDirPath // Path relative to repo root for filtering

	// File Selection View state
	FolderTree      FileDispositionTreeModel // Hierarchical tree view of files
	FileContent     FileContentModel         // File content display (for file selection)
	FilesTable      FilesTableModel          // Files table display (for directory selection)
	IsDirectoryView bool                     // true = showing directory table, false = showing file content
	ModuleScoped    bool                     // true = module-scoped, false = full-repo

	// Takes View state
	Takes      *gomcfg.PlanTakes
	ActiveTake int // 1-based index

	// ChangeSets state
	ActiveCS   int // Active ChangeSet index (1-based)
	ChangeSets []*ChangeSet

	// Files View state (hunk assignment)
	Files []File

	// File content cache: path → File (Content and YOffset populated)
	FileCache map[dt.RelFilepath]*File

	// Git status cache: path → GitFileStatus
	GitStatusCache map[dt.RelFilepath]gitutils.GitFileStatus

	// UI state
	ViewMode       ViewMode
	FocusPane      Pane
	Width          int
	Height         int
	Writer         cliutil.Writer
	Err            error
	ViewportsReady bool // Track if viewports have been initialized
}

// Ensure EditorState implements tea.Model interface
var _ tea.Model = (*EditorState)(nil)

// Init implements tea.Model - kicks off file loading
func (es EditorState) Init() tea.Cmd {
	// Return a Cmd that loads files asynchronously
	return func() tea.Msg {
		ctx := context.Background()
		files, err := es.loadSelectedFiles(ctx)
		return filesLoadedMsg{files: files, err: err}
	}
}

// Update implements tea.Model - dispatches to view-specific update handlers
func (es EditorState) Update(msg tea.Msg) (m tea.Model, c tea.Cmd) {
	var ctx context.Context

	// Create context for this update
	ctx = context.Background()

	// Dispatch based on current view mode
	switch es.ViewMode {
	case FileSelectionView:
		m, c = es.updateFileSelectionView(ctx, msg)
	case TakesView:
		// TODO: Implement takes view update
		c = nil
	case FilesView:
		// TODO: Implement files view update
		c = nil
	}

	return m, c
}

// View implements tea.Model - dispatches to view-specific renderers
func (es EditorState) View() string {
	// Dispatch based on current view mode
	switch es.ViewMode {
	case FileSelectionView:
		return es.renderFileSelectionView()
	case TakesView:
		// TODO: Implement takes view rendering
		return "Takes View - Not Implemented"
	case FilesView:
		// TODO: Implement files view rendering
		return "Files View - Not Implemented"
	default:
		return "Unknown View Mode"
	}
}

// initFileSelectionView initializes the file selection view state
// Returns the initialized state (following immutable model pattern)
// Note: Child models are not initialized here - they're initialized on first WindowSizeMsg
func (es EditorState) initFileSelectionView(_ context.Context) (EditorState, error) {
	es.ModuleScoped = true
	es.ViewportsReady = false // Will be set true after first WindowSizeMsg
	es.FocusPane = LeftPane
	es.FileCache = make(map[dt.RelFilepath]*File)
	return es, nil
}

// updateFileSelectionView handles updates for file selection view
// Returns updated model and command (following BubbleTea's immutable Elm architecture)
//
//goland:noinspection GoAssignmentToReceiver
func (es EditorState) updateFileSelectionView(ctx context.Context, msg tea.Msg) (_ EditorState, cmd tea.Cmd) {
	var cmds []tea.Cmd
	var err error

	switch msg := msg.(type) {
	case filesLoadedMsg:
		// Handle file loading result
		if msg.err != nil {
			es.Err = msg.err
			return es, nil
		}

		if len(msg.files) == 0 {
			es.Err = NewErr(ErrNoChangedFiles)
			es.ViewportsReady = true
			return es, nil
		}

		es, err = es.loadFiles(ctx, msg.files)
		if err != nil {
			es.Err = err
		}
		return es, nil

	case tea.WindowSizeMsg:
		es.Width = msg.Width
		es.Height = msg.Height

		// Only resize if components exist (after files loaded)
		if es.ViewportsReady {
			layout := es.Layout()
			es.FolderTree = es.FolderTree.SetSize(layout.LeftPaneWidth(), layout.PaneInnerHeight())
			es.FileContent = es.FileContent.SetSize(layout.RightPaneInnerWidth(), layout.PaneInnerHeight())
			if es.IsDirectoryView {
				es.FilesTable = es.FilesTable.SetSize(layout.RightPaneInnerWidth(), layout.PaneHeight())
			}
		}

		return es, nil

	case tea.KeyMsg:
		ms := msg.String()

		// Global keys (always work)
		switch ms {
		case "q", "ctrl+c":
			return es, tea.Quit
		}

		// Check if tree is empty (no files to commit)
		selectedFile := es.FolderTree.SelectedFile()
		isEmpty := selectedFile == nil || selectedFile.IsEmpty()

		// View-specific keys (disabled when tree is empty)
		if !isEmpty {
			fd := FileDisposition(ms[0])
			if IsFileDisposition(fd) {
				// Disposition changes
				return es.setDisposition(fd), nil
			}

			switch ms {
			case "tab":
				// Save scroll position before switching
				selectedFile := es.FolderTree.SelectedFile()
				if selectedFile != nil && es.FocusPane == RightPane {
					es = es.withUpdatedFileCache(selectedFile.Path, es.FileContent.YOffset())
				}

				// Rotate right: Left → Right → Left
				if es.FocusPane == LeftPane {
					es.FocusPane = RightPane
				} else {
					es.FocusPane = LeftPane
				}
				return es, nil

			case "shift+tab":
				// Save scroll position before switching
				selectedFile := es.FolderTree.SelectedFile()
				if selectedFile != nil && es.FocusPane == RightPane {
					es = es.withUpdatedFileCache(selectedFile.Path, es.FileContent.YOffset())
				}

				// Rotate left: Right → Left → Right
				if es.FocusPane == RightPane {
					es.FocusPane = LeftPane
				} else {
					es.FocusPane = RightPane
				}
				return es, nil
			}
		}

		// Module toggle works both when empty and not empty
		switch ms {
		case "m": // Toggle module-scoped / full-repo
			es.ModuleScoped = !es.ModuleScoped

			// Clear caches (file paths change with module scope)
			es.FileCache = make(map[dt.RelFilepath]*File)
			es.GitStatusCache = nil // Clear git status cache

			// Reset focus to tree
			es.FocusPane = LeftPane

			var err error
			files, err := es.loadSelectedFiles(ctx)
			if err != nil {
				es.Err = err
				return es, nil
			}
			// Recreate tree with new file list
			layout := es.Layout()
			es.FolderTree = NewFileDispositionTreeModel(files, layout.LeftPaneWidth(), layout.PaneInnerHeight())
			return es, nil
		}
	}

	// Delegate based on focused pane
	switch es.FocusPane {
	case LeftPane:
		// Tree has focus - handle navigation
		es.FolderTree, cmd = es.FolderTree.Update(msg)
		cmds = append(cmds, cmd)

		// Update file content or directory table when tree selection changes
		selectedFile := es.FolderTree.SelectedFile()
		if selectedFile != nil {
			selectedNode := es.FolderTree.SelectedNode()
			if selectedNode != nil && !selectedNode.HasChildren() {
				// File selected - show file content
				es.IsDirectoryView = false

				// Load with cache (loadFileContent() now handles caching)
				actualPath := es.getActualPath(selectedFile.Path)
				content, yOffset, err := es.loadFileContent(actualPath)
				if err != nil {
					content = fmt.Sprintf("Error loading file:\n%v", err)
					yOffset = 0
				}
				es.FileContent = es.FileContent.SetContent(content, yOffset)
			} else {
				var summary DirSummary
				var dir *Directory
				var layout FileDispositionLayout

				// Directory selected - show directory table
				es.IsDirectoryView = true

				// Get child files from tree
				childFiles := getChildFiles(selectedNode)

				// Get or load git status
				es.GitStatusCache, err = es.gitStatusMap(ctx)
				if err != nil {
					es.Err = err
					goto end
				}

				// Batch load metadata for child files
				_ = batchLoadMetadata(childFiles, es.UserRepo.Root)

				// Enrich with git status
				for _, file := range childFiles {
					enrichWithGitStatus(file, es.GitStatusCache)
				}

				// Calculate summary
				summary = calculateDirSummary(childFiles)

				// Create directory object
				dir = &Directory{
					Path:    dt.RelDirPath(selectedFile.Path),
					Files:   childFiles,
					Summary: &summary,
				}

				// Create/update table
				layout = es.Layout()
				es.FilesTable = NewFilesTableModel(dir, layout.RightPaneInnerWidth(), layout.PaneHeight())
			end:
				return es, cmd
			}
		}

	case RightPane:
		// Right pane has focus - delegate to either file content or directory table
		if es.IsDirectoryView {
			// Directory table has focus - handle navigation and disposition changes
			es.FilesTable, cmd = es.FilesTable.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			// File content has focus - handle scrolling
			es.FileContent, cmd = es.FileContent.Update(msg)
			cmds = append(cmds, cmd)

			// Update cache with current scroll position
			selectedFile := es.FolderTree.SelectedFile()
			if selectedFile != nil {
				es = es.withUpdatedFileCache(selectedFile.Path, es.FileContent.YOffset())
			}
		}

	default:
		// Stop linting from complaining
	}

	return es, tea.Batch(cmds...)
}

// setDisposition applies disposition to the selected node
func (es EditorState) setDisposition(disp FileDisposition) EditorState {
	selectedNode := es.FolderTree.SelectedNode()
	if selectedNode == nil {
		return es
	}

	// Apply disposition to node (and recursively to children if folder)
	applyDispositionToNode(selectedNode, disp)

	return es
}

// applyDispositionToNode recursively applies disposition to a node and its children
func applyDispositionToNode(node *FileDispositionNode, disp FileDisposition) {
	// Update this node
	node.Data().Disposition = disp
	// If folder, recursively update all children
	if node.HasChildren() {
		for _, child := range node.Children() {
			applyDispositionToNode(child, disp)
		}
	}
}

// renderFileSelectionView renders the file selection view
func (es EditorState) renderFileSelectionView() string {
	if !es.ViewportsReady {
		return "Initializing..."
	}

	// Check for errors
	if es.Err != nil {
		if errors.Is(es.Err, ErrNoChangedFiles) {
			return "No changed files to display.\n\nPress q to quit."
		}
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", es.Err)
	}

	var sb strings.Builder

	// Header
	scope := fmt.Sprintf("Module=%s", renderRGBColor(es.ModuleDir.ToTilde(dt.OrFullPath), GreenColor))
	if !es.ModuleScoped {
		scope = fmt.Sprintf("Repo=%s", es.UserRepo.Root)
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		Render("Files to Commit: " + scope)

	// Create layout from current state
	layout := es.Layout()

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

	// Resize components using layout
	es.FolderTree = es.FolderTree.SetSize(layout.LeftPaneWidth(), layout.PaneInnerHeight())
	es.FileContent = es.FileContent.SetSize(layout.RightPaneInnerWidth(), layout.PaneInnerHeight())

	if es.IsDirectoryView {
		es.FilesTable = es.FilesTable.SetSize(layout.RightPaneInnerWidth(), layout.PaneHeight())
	}

	// Calculate border colors based on focus
	leftBorderColor := GrayColor
	rightBorderColor := GrayColor

	if es.FocusPane == LeftPane {
		leftBorderColor = CyanColor
	} else if es.FocusPane == RightPane {
		rightBorderColor = CyanColor
	}

	// Create styled panes
	basePaneStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		Height(layout.PaneInnerHeight())

	leftPane := basePaneStyle.
		PaddingLeft(1).
		PaddingRight(1).
		BorderForeground(lipgloss.Color(leftBorderColor)).
		Render(es.FolderTree.View())

	// Render right pane based on view type
	var rightPane string
	if es.IsDirectoryView {
		// Table already has its own borders - render directly
		es.FilesTable = es.FilesTable.SetBorderColor(rightBorderColor)
		rightPane = es.FilesTable.View()
	} else {
		// File content needs a pane wrapper with borders
		rightPane = basePaneStyle.
			Width(layout.RightPaneWidth()).
			PaddingLeft(1).
			BorderForeground(lipgloss.Color(rightBorderColor)).
			Render(es.FileContent.View())
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	sb.WriteString(header)
	sb.WriteString("\n")
	sb.WriteString(body)
	sb.WriteString("\n")
	sb.WriteString(footer)

	return sb.String()
}

// loadSelectedFiles loads changed files, optionally filtered to module scope
func (es EditorState) loadSelectedFiles(ctx context.Context) (files []File, err error) {
	var changedFiles []dt.RelFilepath
	var filter func(dt.RelFilepath) bool

	// Create filter if module-scoped
	if es.ModuleScoped {
		filter = gompkg.CreateModuleFileFilter(es.ModuleRelPath)
	}

	// Get changed files
	if filter != nil {
		changedFiles, err = es.UserRepo.GetChangedFilesFiltered(ctx, filter)
	} else {
		changedFiles, err = es.UserRepo.GetChangedFiles(ctx)
	}
	if err != nil {
		goto end
	}

	// Convert to File (default to Include)
	files = make([]File, len(changedFiles))
	for i, path := range changedFiles {
		displayPath := path

		// Strip module prefix when module-scoped for cleaner display
		if es.ModuleScoped && es.ModuleRelPath != "" {
			prefix := string(es.ModuleRelPath) + "/"
			pathStr := string(path)
			if strings.HasPrefix(pathStr, prefix) {
				displayPath = dt.RelFilepath(strings.TrimPrefix(pathStr, prefix))
			}
		}

		files[i] = File{
			Path:        displayPath,
			Disposition: CommitDisposition,
			Content:     "", // Will be loaded on demand
		}
	}

end:
	return files, err
}

// getActualPath restores the full repo-relative path from display path
// When module-scoped, display paths have module prefix stripped, so we restore it
func (es EditorState) getActualPath(displayPath dt.RelFilepath) dt.RelFilepath {
	if !es.ModuleScoped || es.ModuleRelPath == "" {
		return displayPath
	}

	// Restore module prefix
	return dt.RelFilepath(string(es.ModuleRelPath) + "/" + string(displayPath))
}

// loadFileContent loads file content, using cache if available
func (es EditorState) loadFileContent(path dt.RelFilepath) (content string, yOffset int, err error) {
	var filepath dt.Filepath
	var bytes []byte
	var actualPath dt.RelFilepath
	var cached *File
	var ok bool
	var pathStr string
	var parts []string

	// Check cache first
	cached, ok = es.FileCache[path]
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
	filepath = dt.FilepathJoin(es.UserRepo.Root, actualPath)

	// Read file
	bytes, err = filepath.ReadFile()
	if err != nil {
		err = NewErr(ErrGit, filepath.ErrKV(), err)
		goto end
	}

	content = string(bytes)

	// Cache it
	es.FileCache[path] = &File{
		Path:    path,
		Content: content,
		YOffset: 0,
	}

	yOffset = 0

end:
	return content, yOffset, err
}

// withUpdateFileCache updates the cached scroll position for a file
func (es EditorState) withUpdatedFileCache(path dt.RelFilepath, yOffset int) EditorState {
	var updated File
	var newCache map[dt.RelFilepath]*File

	cached, ok := es.FileCache[path]
	if !ok {
		goto end
	}
	// Create new File with updated YOffset
	updated = *cached
	updated.YOffset = yOffset

	// Create new map with updated entry
	newCache = make(map[dt.RelFilepath]*File, len(es.FileCache))
	for k, v := range es.FileCache {
		newCache[k] = v
	}
	newCache[path] = &updated

	// Return new EditorState with new cache
	es.FileCache = newCache
end:
	return es
}

// gitStatusMap returns the cached git status map, loading it if necessary.
// Runs git status --porcelain and caches the result in EditorState.GitStatusCache.
func (es EditorState) gitStatusMap(ctx context.Context) (m gitutils.StatusMap, err error) {
	// Return cached if available
	if es.GitStatusCache != nil {
		m = es.GitStatusCache
		goto end
	}

	// Load git status
	m, err = es.UserRepo.StatusMap(ctx, &gitutils.StatusArgs{
		HumanReadable: false,
	})

end:
	return m, err
}

func (es EditorState) loadFiles(ctx context.Context, files []File) (_ EditorState, err error) {
	var selectedNode *FileDispositionNode
	var selectedFile *File
	var childFiles []*File
	var gitStatusMap map[dt.RelFilepath]gitutils.GitFileStatus
	var summary DirSummary
	var dir *Directory

	// Create tree with loaded files
	layout := es.Layout()
	if len(files) == 0 {
		es.FolderTree = NewEmptyFileDispositionTreeModel(
			"No changed files available for commit in current module.\n\n"+
				"Use 'm' to toggle between module and repository scope.",
			layout.LeftPaneWidth(),
			layout.PaneInnerHeight(),
		)
		es.FileContent = NewFileContentModel(layout.RightPaneInnerWidth(), layout.PaneInnerHeight())
		es.ViewportsReady = true
		// Don't return error - this is a valid state
		goto end
	}

	es.FolderTree = NewFileDispositionTreeModel(files, layout.LeftPaneWidth(), layout.PaneInnerHeight())
	es.FileContent = NewFileContentModel(layout.RightPaneInnerWidth(), layout.PaneInnerHeight())

	// Load initial file content or directory table
	selectedFile = es.FolderTree.SelectedFile()
	if selectedFile == nil {
		goto end
	}
	selectedNode = es.FolderTree.SelectedNode()

	if !selectedNode.HasChildren() {
		// File selected - load content
		es.IsDirectoryView = false
		actualPath := es.getActualPath(selectedFile.Path)
		content, yOffset, err := es.loadFileContent(actualPath)
		if err != nil {
			content = fmt.Sprintf("Error loading file:\n%v", err)
			yOffset = 0
		}
		es.FileContent = es.FileContent.SetContent(content, yOffset)
		goto end
	}

	if selectedNode == nil {
		goto end
	}

	// Directory selected - create table
	es.IsDirectoryView = true
	childFiles = getChildFiles(selectedNode)
	gitStatusMap, err = es.gitStatusMap(ctx)
	if err != nil {
		goto end
	}

	_ = batchLoadMetadata(childFiles, es.UserRepo.Root)
	for _, file := range childFiles {
		enrichWithGitStatus(file, gitStatusMap)
	}
	summary = calculateDirSummary(childFiles)
	dir = &Directory{
		Path:    dt.RelDirPath(selectedFile.Path),
		Files:   childFiles,
		Summary: &summary,
	}
	es.FilesTable = NewFilesTableModel(dir, layout.RightPaneInnerWidth(), layout.PaneHeight())

end:
	if err == nil {
		es.ViewportsReady = true
	}
	return es, err
}

func (es EditorState) Layout() FileDispositionLayout {
	treeWidth := 0
	if es.ViewportsReady {
		treeWidth = es.FolderTree.LayoutWidth()
	}
	return NewFileDispositionLayout(es.Width, es.Height, treeWidth)
}

// filesLoadedMsg is sent when files are loaded asynchronously
type filesLoadedMsg struct {
	files []File
	err   error
}
