package gomtui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/bubbletree"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
	"github.com/mikeschinkel/gomion/gommod/gomcfg"
	"github.com/mikeschinkel/gomion/gommod/gompkg"
	"go.dalton.dog/bubbleup"
	"golang.org/x/term"
)

type FileCache map[dt.RelFilepath]*bubbletree.File

// EditorState is the main bubbletea model for the GRU staging editor
type EditorState struct {
	// Repository state
	CachedRepo *gitutils.CachedWorktree
	UserRepo   *gitutils.Repo
	ModuleDir  dt.DirPath

	context context.Context // This is not idiomatic for Go, but necessary for a BubbleTea app

	// File Selection View state
	layout FileDispositionModel

	// Takes View state
	Takes      *gomcfg.PlanTakes
	ActiveTake int // 1-based index

	// ChangeSets state
	ActiveCS   int // Active ChangeSet index (1-based)
	ChangeSets []*ChangeSet

	// Files View state (hunk assignment)
	fileSource *FileSource

	// File content cache: path → File (Content and YOffset populated)
	FileCache FileCache

	// Git status cache: path → FileStatus
	GitStatusCache gitutils.StatusMap

	// Disposition map: path → FileDisposition (pointer for efficient in-place updates)
	dispositions     map[dt.RelFilepath]FileDisposition
	dispositionCache DispositionsCache

	// Commit plan persistence (auto-save)
	saveSeq       int           // Auto-save sequence number (incremented on each disposition change)
	activeSaveSeq int           // Active save operation sequence (for staleness detection)
	saveDebounce  time.Duration // Debounce period (default 3 seconds)
	saving        bool          // UI indicator for save in progress

	// Alert system for notifications
	Alert bubbleup.AlertModel // Alert overlay for save notifications

	// UI state
	ViewMode ViewMode
	Width    int
	Height   int
	Writer   cliutil.Writer
	Err      error
}

type EditorStateArgs struct {
	Writer   cliutil.Writer
	UserRepo *gitutils.Repo
}

func NewEditorState(moduleDir dt.DirPath, args EditorStateArgs) EditorState {
	// Initialize auto-save debounce (3 seconds)
	return EditorState{
		ModuleDir:    moduleDir,
		UserRepo:     args.UserRepo,
		Writer:       args.Writer,
		ViewMode:     FileSelectionView, // Start in File Selection View
		saveDebounce: 3 * time.Second,
		dispositions: make(map[dt.RelFilepath]FileDisposition),
		Alert: bubbleup.NewAlertModel(50, false, 3*time.Second).
			WithMinWidth(10).
			WithUnicodePrefix().
			WithPosition(bubbleup.TopRightPosition).
			WithAllowEscToClose(),
		FileCache: make(FileCache),
	}
}

// Ensure EditorState implements tea.Model interface
var _ tea.Model = (*EditorState)(nil)

func (es EditorState) Initialize() (_ EditorState, err error) {
	var relPath dt.PathSegments
	relPath, err = es.ModuleDir.Rel(es.UserRepo.Root)
	if err != nil {
		goto end
	}
	es.fileSource = NewFileSource(relPath)
end:
	return es, err
}

func (es EditorState) HasDispositions() bool {
	return len(es.dispositions) > 0
}
func (es EditorState) NumDispositions() int {
	return len(es.dispositions)
}

func (es EditorState) FileSource() (fs *FileSource, err error) {
	if es.fileSource.HasFiles() {
		goto end
	}
	err = es.fileSource.LoadChangedRepoFiles(es.Context(), es.UserRepo)
end:
	return es.fileSource, err
}

func (es EditorState) Context() context.Context {
	if es.context == nil {
		es.context = context.Background()
	}
	return es.context
}

// loadFilesCmd returns the command to load repo files into its filesource
func (es EditorState) loadFilesCmd() func() tea.Msg {
	return func() tea.Msg {
		var fs *FileSource
		var err error
		fs, err = es.FileSource()
		return filesLoadedMsg{
			FileSource: fs,
			err:        err,
		}
	}
}

// Init implements tea.Model - kicks off file loading and commit plan loading
func (es EditorState) Init() tea.Cmd {
	// Load files, commit plan and alert in parallel
	// Clear caches (file paths change with module scope)
	es.FileCache = make(FileCache)
	es.GitStatusCache = nil // Clear git status cache
	return tea.Batch(
		es.loadFilesCmd(),
		es.commitPlanCmd().loadCmd(),
		es.Alert.Init(),
	)
}

// ModuleRelPath calculates the relative path from repo root to module directory
func (es EditorState) ModuleRelPath() (relPath dt.RelDirPath) {
	//func calculateModuleRelPath(repoRoot dt.DirPath, moduleDir dt.DirPath) dt.RelDirPath {
	var err error

	if es.ModuleDir == es.UserRepo.Root {
		// If module dir equals repo root, use "."
		goto end
	}

	// Calculate relative path from repo root to module
	relPath, err = es.ModuleDir.Rel(es.UserRepo.Root)
	if err != nil {
		es.Err = err
		goto end
	}

end:
	return relPath
}

// Disposition returns the disposition for a file path.
// Returns UnspecifiedDisposition if no explicit disposition has been set.
func (es EditorState) Disposition(path dt.RelFilepath) (disp FileDisposition) {
	var ok bool
	// Check exact match, or return default
	disp, ok = (es.dispositions)[path]
	if !ok {
		// Default: requires user action
		disp = UnspecifiedDisposition
	}
	return disp
}

type DispositionFunc func(path dt.RelFilepath) FileDisposition

// DispositionFunc returns a func that provides the disposition for a file path.
func (es EditorState) DispositionFunc() DispositionFunc {
	return es.Disposition
}

// SetDisposition updates the disposition for a file path (mutates map in-place).
// The map is accessed via pointer, so changes are visible to all callbacks.
// For recursive updates (directories), handle at call site.
func (es EditorState) SetDisposition(path dt.RelFilepath, disp FileDisposition) EditorState {
	// Update map in-place through pointer
	es.dispositions[path] = disp
	return es
}

// setDispositionCallback returns a callback for FileDispositionModel to notify of disposition changes
func (es EditorState) setDispositionCallback() SetDispositionCallback {
	return func(path dt.RelFilepath, disp FileDisposition) tea.Cmd {
		// Update disposition in EditorState
		es.SetDisposition(path, disp)
		// Return command to schedule save
		return scheduleSaveCmd
	}
}

// commitPlanCmd returns a fresh commitPlanCmd with current EditorState data
func (es EditorState) commitPlanCmd() commitPlanCmd {
	// Convert dispositions map to string map for persistence
	planMap := make(gompkg.CommitPlanMap, es.NumDispositions())
	for path, dispLabel := range es.dispositionCache.LabelsMap() {
		// Store as lowercase label (e.g., "commit", "omit", "ignore", "exclude")
		planMap[path] = dispLabel
	}

	// Determine scope
	scope := gompkg.RepoScope
	if !es.layout.RepoScoped {
		scope = gompkg.ModuleScope
	}

	return commitPlanCmd{
		RepoRoot:   es.UserRepo.Root,
		ModulePath: es.ModuleRelPath(),
		Scope:      scope,
		CommitPlan: planMap,
	}
}

// LoadCommitPlanData populates dispositions from loaded plan
func (es EditorState) LoadCommitPlanData(plan *gompkg.CommitPlan) EditorState {
	if plan == nil {
		goto end
	}

	// Parse string values back to FileDisposition
	for path, dispStr := range plan.CommitPlan {
		disp, err := ParseFileDisposition(string(dispStr))
		if err != nil {
			// Log error but continue with other files
			// TODO: Add logger call here
			continue
		}
		es.SetDisposition(path, disp)
	}
end:
	return es
}

// scheduleSave schedules save after debounce period (returns command)
func (es EditorState) scheduleSave() (EditorState, tea.Cmd) {
	// Bump sequence
	es.saveSeq++
	seq := es.saveSeq

	// Return a command that waits and then sends SaveMsgType
	cmd := func() tea.Msg {
		time.Sleep(es.saveDebounce)
		return commitPlanMsg{
			msgType: SaveMsgType,
			seq:     seq,
		}
	}

	return es, cmd
}

// Update implements tea.Model - dispatches to view-specific update handlers
func (es EditorState) Update(msg tea.Msg) (m tea.Model, c tea.Cmd) {
	var ctx context.Context

	// Create context for this update
	ctx = context.Background()

	// Dispatch based on current view mode
	switch es.ViewMode {
	case FileSelectionView:
		m, c = es.updateFileSelection(ctx, msg)
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

func (es EditorState) CreateLayout() FileDispositionModel {
	var layout FileDispositionModel
	var fs *FileSource
	var err error

	// Create tree, table, content with ACTUAL dimensions No guessing, no defaults,
	// no conditional logic This runs ONCE when both prerequisites are met Use
	// terminalWidth() and terminalHeight() methods which query actual terminal size
	// if WindowSizeMsg hasn't arrived yet (prevents creating table with width=0)
	fs, err = es.FileSource()
	if err != nil {
		es.Err = err
		goto end
	}

	layout = NewFileDispositionModel(FileDispositionModelArgs{
		FileSource:      fs,
		UserRepo:        es.UserRepo,
		ModuleDir:       es.ModuleDir,
		Width:           es.terminalWidth(),
		Height:          es.terminalHeight(),
		DispositionFunc: es.DispositionFunc(),
		SetDisposition:  es.setDispositionCallback(),
		RepoScoped:      es.layout.RepoScoped,
	})

end:
	return layout
}
func (es EditorState) nonZeroInt(value int, def func() int) int {
	if value == 0 {
		value = def()
	}
	return value
}

// updateFileSelection handles updates for file selection view
// Returns updated model and command (following BubbleTea's immutable Elm architecture)
//
//goland:noinspection GoAssignmentToReceiver
func (es EditorState) updateFileSelection(ctx context.Context, msg tea.Msg) (_ EditorState, cmd tea.Cmd) {
	// Delegate all messages to Alert first (for ticks and ESC)
	var alertCmd tea.Cmd
	alertModel, alertCmd := es.Alert.Update(msg)
	es.Alert = alertModel.(bubbleup.AlertModel)

	// Handle different message types
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update EditorState dimensions
		es.Width = msg.Width
		es.Height = msg.Height

		// If we have files loaded, create layout if needed
		if es.fileSource != nil && !es.layout.Initialized() {
			return es, createLayoutCmd
		}

		// If layout exists, delegate resize to it
		if es.layout.Initialized() {
			updatedModel, layoutCmd := es.layout.Update(resizeLayoutMsg{
				Width:  msg.Width,
				Height: msg.Height,
			})
			es.layout = updatedModel.(FileDispositionModel)
			return es, tea.Batch(alertCmd, layoutCmd)
		}

		return es, alertCmd

	case filesLoadedMsg:
		es.fileSource = msg.FileSource
		if es.hasDimensions() {
			return es, createLayoutCmd
		}
		return es, alertCmd

	case createLayoutMsg:
		// Will be called after both WindowSizeMsg and filesLoadedMsg
		es.layout = es.CreateLayout()
		// Send initial resize to layout
		updatedModel, layoutCmd := es.layout.Update(resizeLayoutMsg{
			Width:  es.Width,
			Height: es.Height,
		})
		es.layout = updatedModel.(FileDispositionModel)
		return es, tea.Batch(alertCmd, layoutCmd)

	case changeDispositionMsg:
		// Handle disposition changes from table/tree view
		// If node is provided (directory in tree view), cascade to all descendants
		// Otherwise just update the single path (file in tree view or table view)
		if msg.Node != nil {
			// Directory selected - cascade to all descendants
			paths := GetAllDescendantPaths(msg.Node)
			for _, path := range paths {
				es = es.SetDisposition(path, msg.Disposition)
			}
		} else {
			// File selected - just update the single path
			es = es.SetDisposition(msg.Path, msg.Disposition)
		}
		// Batch both save and table refresh for async pattern
		return es, tea.Batch(scheduleSaveCmd, refreshTableCmd)

	case commitPlanMsg:
		var planCmd tea.Cmd
		es, planCmd = es.handleCommitPlanMsg(msg)
		return es, tea.Batch(alertCmd, planCmd)

	case scheduleSaveMsg:
		var saveCmd tea.Cmd
		es, saveCmd = es.scheduleSave()
		return es, tea.Batch(alertCmd, saveCmd)

	case tea.KeyMsg:
		ms := msg.String()

		// Global keys (always work)
		if ms == "q" || ms == "ctrl+c" {
			return es, tea.Quit
		}

		// Manual save (Ctrl+S or Cmd+S)
		if ms == "ctrl+s" {
			es.saving = true
			es.saveSeq++
			es.activeSaveSeq = es.saveSeq
			return es, tea.Batch(alertCmd, es.commitPlanCmd().saveCmd(es.activeSaveSeq))
		}

		// Module toggle works both when empty and not empty
		if ms == "m" {
			var toggleCmd tea.Cmd
			es, toggleCmd = es.handleModuleToggle(ctx)
			return es, tea.Batch(alertCmd, toggleCmd)
		}
	}

	// Delegate to FileDispositionModel for all other messages
	if es.layout.Initialized() {
		updatedModel, layoutCmd := es.layout.Update(msg)
		es.layout = updatedModel.(FileDispositionModel)
		return es, tea.Batch(alertCmd, layoutCmd)
	}

	return es, alertCmd
}

// setDisposition applies disposition to the selected node
func (es EditorState) setDisposition(disp FileDisposition, node ...*FileDispositionNode) EditorState {
	var selectedNode *FileDispositionNode
	var file *bubbletree.File
	var path dt.RelFilepath

	switch len(node) {
	case 0:
		selectedNode = es.layout.FolderTree.FocusedNode()
	default:
		path = node[0].Data().Path
	}
	if selectedNode == nil {
		goto end
	}
	file = selectedNode.Data()
	path = file.Path

	// Set disposition for this path (file or directory)
	es = es.SetDisposition(path, disp)

	// Apply disposition recursively to all children if folder
	if selectedNode.HasChildren() {
		for _, child := range selectedNode.Children() {
			es = es.setDisposition(disp, child)
		}
	}
end:
	return es
}

// renderFileSelectionView renders the file selection view
func (es EditorState) renderFileSelectionView() string {
	// Check if layout is initialized before delegating
	if !es.layout.Initialized() {
		return "Initializing..."
	}

	// Delegate to FileDispositionModel for rendering
	view := es.layout.View()

	// Overlay alert on top of all content (MUST be last)
	return es.Alert.Render(view)
}

// loadFileContent loads file content, using cache if available
func (es EditorState) loadFileContent(path dt.RelFilepath) (content string, yOffset int, err error) {
	var filepath dt.Filepath
	var bytes []byte
	var actualPath dt.RelFilepath
	var cached *bubbletree.File
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
	es.FileCache[path] = bubbletree.NewFile(path, content).SetData(NewFileData())
	yOffset = 0

end:
	return content, yOffset, err
}

// withUpdateFileCache updates the cached scroll position for a file
func (es EditorState) withUpdatedFileCache(path dt.RelFilepath, yOffset int) EditorState {
	var updated bubbletree.File
	var newCache FileCache

	cached, ok := es.FileCache[path]
	if !ok {
		goto end
	}
	// Create new File with updated YOffset
	updated = *cached
	updated.YOffset = yOffset

	// Create new map with updated entry
	newCache = make(FileCache, len(es.FileCache))
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

func (es EditorState) loadFiles(ctx context.Context, files []bubbletree.File) (_ EditorState, err error) {
	var selectedNode *FileDispositionNode
	var selectedFile *bubbletree.File
	var childFiles []*bubbletree.File
	var gitStatusMap gitutils.StatusMap
	var dir Directory

	//// Create tree with loaded files
	//layout := es.CreateLayout()
	//es.layout.FileContent = NewFileContentModel(layout.RightPaneInnerWidth(), layout.PaneInnerHeight())
	//
	//if len(files) == 0 {
	//	es.layout.FolderTree = NewEmptyFileDispositionTreeModel("No changed files available for commit in current module.\n\nUse 'm' to toggle between module and repository scope.",
	//		layout.PaneInnerHeight(),
	//		es.DispositionFunc(),
	//	)
	//	//es.ViewportsReady = true
	//	// Don't return error - this is a valid state
	//	goto end
	//}
	//
	//// Store files for later use (rebuilding tree after disposition changes)
	//es.Files = files
	//
	//es.layout.FolderTree = NewFileDispositionTreeModel(es.fileSource, layout.PaneInnerHeight(), es.DispositionFunc())
	//
	// Load initial file content or directory table
	selectedFile = es.layout.FolderTree.SelectedFile()
	if selectedFile == nil {
		goto end
	}
	selectedNode = es.layout.FolderTree.FocusedNode()

	if !selectedNode.HasChildren() {
		// File selected - load content
		es.layout.IsDirectoryView = false
		content, yOffset, err := es.loadFileContent(selectedFile.Path)
		if err != nil {
			content = fmt.Sprintf("Error loading file:\n%v", err)
			yOffset = 0
		}
		es.layout.FileContent = es.layout.FileContent.SetContent(content, yOffset)
		goto end
	}

	if selectedNode == nil {
		goto end
	}

	// Directory selected - create table
	es.layout.IsDirectoryView = true
	childFiles = GetNodeChildData[bubbletree.File](selectedNode)
	gitStatusMap, err = es.gitStatusMap(ctx)
	if err != nil {
		es.Err = err
		goto end
	}

	err = batchLoadMeta(childFiles, es.UserRepo.Root)
	if err != nil {
		es.Err = err
		goto end
	}

	for _, file := range childFiles {
		gitStatusMap.EnsureFileStatus(file.Path)
	}
	dir = Directory{
		Path:  dt.RelDirPath(selectedFile.Path),
		Files: childFiles,
	}
	es.layout.FilesTable = NewFilesTableModel(dir,
		es.DispositionFunc(),
		es.layout.RightPaneInnerWidth(),
		es.layout.PaneHeight(),
	)

end:
	return es, err
}

// handleFilesLoaded processes the filesLoadedMsg after async file loading
func (es EditorState) handleFilesLoaded(ctx context.Context, msg filesLoadedMsg) (EditorState, tea.Cmd) {
	var err error

	// Handle file loading result
	if msg.err != nil {
		es.Err = msg.err
		goto end
	}

	if !msg.HasFiles() {
		es.Err = NewErr(ErrNoChangedFiles)
		goto end
	}

	es.fileSource = msg.FileSource

	es, err = es.loadFiles(ctx, msg.Files())
	if err != nil {
		es.Err = err
	}
end:
	return es, nil
}

// handleCommitPlanMsg handles all commit plan save/load messages
func (es EditorState) handleCommitPlanMsg(msg commitPlanMsg) (EditorState, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.msgType {
	case SaveMsgType:
		// Debounce timer fired - ignore if stale
		if msg.seq != es.saveSeq {
			goto end
		}

		// Execute save (non-blocking)
		es.saving = true
		es.activeSaveSeq = es.saveSeq
		cmd = es.commitPlanCmd().saveCmd(es.activeSaveSeq)

	case SaveCompleteMsgType:
		// Ignore stale results
		if msg.seq != es.activeSaveSeq {
			goto end
		}

		es.saving = false

		// Display alert based on save result
		if msg.err != nil {
			cmd = es.Alert.NewAlertCmd(bubbleup.ErrorKey,
				fmt.Sprintf("Save failed: %v", msg.err))
		} else {
			cmd = es.Alert.NewAlertCmd(bubbleup.InfoKey, "Commit plan saved")
		}

	case LoadCompleteMsgType:
		if msg.err != nil {
			cmd = es.Alert.NewAlertCmd(bubbleup.ErrorKey,
				fmt.Sprintf("Load failed: %v", msg.err))
			goto end
		}

		// Populate dispositions from loaded plan
		es = es.LoadCommitPlanData(msg.plan)
	}

end:
	return es, cmd
}


// handleModuleToggle switches between module-scoped and full-repo view
func (es EditorState) handleModuleToggle(ctx context.Context) (EditorState, tea.Cmd) {

	// Toggle scope
	es.layout.RepoScoped = !es.layout.RepoScoped

	// Call Init() again
	return es, es.Init()
}


// Commit plan persistence messages

// BubbleTeaMsgType identifies the type of commit plan message
type BubbleTeaMsgType int

const (
	UnspecifiedMsgType BubbleTeaMsgType = iota
	LoadCompleteMsgType
	SaveCompleteMsgType
	LoadMsgType
	SaveMsgType
)

// commitPlanCmd encapsulates parameters for commit plan save/load commands
type commitPlanCmd struct {
	RepoRoot   dt.DirPath
	ModulePath dt.RelDirPath
	Scope      gompkg.CommitScope
	CommitPlan gompkg.CommitPlanMap
}

// SaveCmd runs save in background (async I/O)
func (cmd commitPlanCmd) saveCmd(seq int) tea.Cmd {
	return func() tea.Msg {
		plan := &gompkg.CommitPlan{
			Version:    1,
			Scope:      cmd.Scope,
			ModulePath: cmd.ModulePath,
			Timestamp:  time.Now(),
			CommitPlan: cmd.CommitPlan,
		}
		return commitPlanMsg{
			msgType: SaveCompleteMsgType,
			seq:     seq,
			err:     plan.Save(cmd.RepoRoot),
		}
	}
}

// LoadCmd runs load in background (async I/O)
func (cmd commitPlanCmd) loadCmd() tea.Cmd {
	return func() tea.Msg {
		plan, err := gompkg.LoadCommitPlan(cmd.RepoRoot)
		return commitPlanMsg{
			msgType: LoadCompleteMsgType,
			plan:    plan,
			err:     err,
		}
	}
}

func (es EditorState) HasFiles() bool {
	return es.fileSource.HasFiles()
}

func (es EditorState) hasDimensions() bool {
	return es.Width > 0 && es.Height > 0
}

// terminalWidth returns the actual terminal width.
// If Width field has been set by WindowSizeMsg, use that.
// Otherwise, query the terminal directly to get current width.
// This handles the case where components are initialized before WindowSizeMsg arrives.
func (es EditorState) terminalWidth() int {
	if es.Width > 0 {
		return es.Width
	}
	// Query actual terminal width from OS
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width == 0 {
		// Fallback to reasonable default if query fails
		return 120
	}
	return width
}

// TerminalHeight returns the actual terminal height.
// If Height field has been set by WindowSizeMsg, use that.
// Otherwise, query the terminal directly to get current height.
// This handles the case where components are initialized before WindowSizeMsg arrives.
func (es EditorState) terminalHeight() int {
	if es.Height > 0 {
		return es.Height
	}
	// Query actual terminal height from OS
	_, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || height == 0 {
		// Fallback to reasonable default if query fails
		return 30
	}
	return height
}
