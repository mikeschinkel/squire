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
	FolderTree   FileDispositionTreeModel // Hierarchical tree view of files
	FileContent  FileContentModel         // File content display
	ModuleScoped bool                     // true = module-scoped, false = full-repo

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

// Init implements tea.Model
func (es EditorState) Init() tea.Cmd {
	return nil
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
func (es EditorState) initFileSelectionView(ctx context.Context) (EditorState, error) {
	es.ModuleScoped = true
	es.ViewportsReady = false // Will be set true after first WindowSizeMsg
	es.FocusPane = LeftPane
	es.FileCache = make(map[dt.RelFilepath]*File)
	return es, nil
}

// updateFileSelectionView handles updates for file selection view
// Returns updated model and command (following BubbleTea's immutable Elm architecture)
func (es EditorState) updateFileSelectionView(ctx context.Context, msg tea.Msg) (EditorState, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		es.Width = msg.Width
		es.Height = msg.Height

		paneWidth := (es.Width - 8) / 2
		paneHeight := es.Height - 6

		// Initialize models on first WindowSizeMsg
		if !es.ViewportsReady {
			var files []File
			var err error

			// Load changed files scoped to module
			files, err = es.loadSelectedFiles(ctx)
			if err != nil {
				es.Err = err
				return es, nil
			}

			// Check if there are no changed files
			if len(files) == 0 {
				// Show message in tree view instead of erroring
				es.FolderTree = NewEmptyFileDispositionTreeModel(
					"No changed files available for commit in current module.\n\n"+
						"Use 'm' to toggle between module and repository scope.",
					paneWidth,
					paneHeight,
				)
				es.FileContent = NewFileContentModel(paneWidth, paneHeight)
				es.ViewportsReady = true
				return es, nil
			}

			// Initialize child models with actual dimensions
			es.FolderTree = NewFileDispositionTreeModel(files, paneWidth, paneHeight)
			es.FileContent = NewFileContentModel(paneWidth, paneHeight)

			// Load initial file content
			if selectedFile := es.FolderTree.GetSelectedFile(); selectedFile != nil {
				if selectedNode := es.FolderTree.GetSelectedNode(); selectedNode != nil && !selectedNode.HasChildren() {
					actualPath := es.getActualPath(selectedFile.Path)
					content, yOffset, err := es.loadFileContent(actualPath)
					if err != nil {
						content = fmt.Sprintf("Error loading file:\n%v", err)
						yOffset = 0
					}
					es.FileContent = es.FileContent.SetContent(content, yOffset)
				}
			}

			es.ViewportsReady = true
		} else {
			// Update existing model dimensions
			es.FolderTree = es.FolderTree.SetSize(paneWidth, paneHeight)
			es.FileContent = es.FileContent.SetSize(paneWidth, paneHeight)
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
		selectedFile := es.FolderTree.GetSelectedFile()
		isEmpty := selectedFile == nil || selectedFile.Path == ""

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
				selectedFile := es.FolderTree.GetSelectedFile()
				if selectedFile != nil && es.FocusPane == RightPane {
					es.updateFileCache(selectedFile.Path, es.FileContent.YOffset())
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
				selectedFile := es.FolderTree.GetSelectedFile()
				if selectedFile != nil && es.FocusPane == RightPane {
					es.updateFileCache(selectedFile.Path, es.FileContent.YOffset())
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

			// Clear cache (file paths change with module scope)
			es.FileCache = make(map[dt.RelFilepath]*File)

			// Reset focus to tree
			es.FocusPane = LeftPane

			var err error
			files, err := es.loadSelectedFiles(ctx)
			if err != nil {
				es.Err = err
				return es, nil
			}
			// Recreate tree with new file list
			paneWidth := (es.Width - 8) / 2
			paneHeight := es.Height - 6
			es.FolderTree = NewFileDispositionTreeModel(files, paneWidth, paneHeight)
			return es, nil
		}
	}

	// Delegate based on focused pane
	switch es.FocusPane {
	case LeftPane:
		// Tree has focus - handle navigation
		es.FolderTree, cmd = es.FolderTree.Update(msg)
		cmds = append(cmds, cmd)

		// Update file content when tree selection changes
		selectedFile := es.FolderTree.GetSelectedFile()
		if selectedFile != nil {
			selectedNode := es.FolderTree.GetSelectedNode()
			if selectedNode != nil && !selectedNode.HasChildren() {
				// Load with cache (loadFileContent() now handles caching)
				actualPath := es.getActualPath(selectedFile.Path)
				content, yOffset, err := es.loadFileContent(actualPath)
				if err != nil {
					content = fmt.Sprintf("Error loading file:\n%v", err)
					yOffset = 0
				}
				es.FileContent = es.FileContent.SetContent(content, yOffset)
			} else {
				// Folder selected - clear content
				es.FileContent = es.FileContent.SetContent("", 0)
			}
		}

	case RightPane:
		// File content has focus - handle scrolling
		es.FileContent, cmd = es.FileContent.Update(msg)
		cmds = append(cmds, cmd)

		// Update cache with current scroll position
		selectedFile := es.FolderTree.GetSelectedFile()
		if selectedFile != nil {
			es.updateFileCache(selectedFile.Path, es.FileContent.YOffset())
		}
	}

	return es, tea.Batch(cmds...)
}

// setDisposition applies disposition to the selected node
func (es EditorState) setDisposition(disp FileDisposition) EditorState {
	selectedNode := es.FolderTree.GetSelectedNode()
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

	menu := fmt.Sprintf("↑/↓:Navigate | ←/→:Expand/Collapse | %s:Commit | %s:Omit | %s:.gitignore | %s:.git/exclude | m:Module/Repo | Enter:Continue | q:Quit",
		CommitDisposition.Key(),
		OmitDisposition.Key(),
		GitIgnoreDisposition.Key(),
		GitExcludeDisposition.Key(),
	)
	// Compact footer with keyboard hints (TODO: add overlay help with '?')
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color(SilverColor)).
		Render(menu)

	// Calculate content height (total height - header - footer - borders)
	contentHeight := es.Height - 4  // 1 for header, 1 for footer, 2 for spacing
	paneHeight := contentHeight - 2 // Subtract border height for viewport

	// Calculate tree content width without artificial minimums
	treeContentWidth := es.FolderTree.GetMaxVisibleWidth()

	// Calculate remaining width for content (account for borders + padding + spacing)
	// Each pane has: 2 for borders + 1 for padding = 3, plus ~4 for spacing
	contentWidth := es.Width - treeContentWidth - 10

	// Resize both viewports to calculated dimensions
	// Use actual content width for tree (no artificial minimum)
	es.FolderTree = es.FolderTree.SetSize(treeContentWidth, paneHeight)
	es.FileContent = es.FileContent.SetSize(contentWidth, paneHeight)

	// Calculate border colors based on focus
	leftBorderColor := GrayColor  // Unfocused
	rightBorderColor := GrayColor // Unfocused

	if es.FocusPane == LeftPane {
		leftBorderColor = CyanColor // Focused
	} else if es.FocusPane == RightPane {
		rightBorderColor = CyanColor // Focused
	}

	// Create styled panes - use same width values for consistency
	basePaneStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		Height(contentHeight)

	leftPane := basePaneStyle.
		PaddingLeft(1).
		PaddingRight(1).
		BorderForeground(lipgloss.Color(leftBorderColor)).
		Render(es.FolderTree.View())

	rightPane := basePaneStyle.
		Width(contentWidth + 3). // +2 for borders, +1 for left padding
		PaddingLeft(1).          // Consistent with left pane
		BorderForeground(lipgloss.Color(rightBorderColor)).
		Render(es.FileContent.View())

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

// updateFileCache updates the cached scroll position for a file
func (es *EditorState) updateFileCache(path dt.RelFilepath, yOffset int) {
	var cached *File
	var ok bool

	cached, ok = es.FileCache[path]
	if ok {
		cached.YOffset = yOffset
	}
}
