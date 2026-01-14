package gomtui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"github.com/mikeschinkel/gomion/gommod/bubbletree"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
	"github.com/mikeschinkel/gomion/gommod/gomcfg"
	"github.com/mikeschinkel/gomion/gommod/gompkg"
	"go.dalton.dog/bubbleup"
)

type FileCache map[dt.RelFilepath]*bubbletree.File

// EditorState is the main bubbletea model for the GRU staging editor
type EditorState struct {
	Logger *slog.Logger

	// Repository state
	UserRepo   *gitutils.Repo
	ModuleDir  dt.DirPath
	modulePath dt.RelDirPath
	commitPlan *gompkg.CommitPlan

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
	fileSource     *FileSource
	GitStatusCache gitutils.StatusMap // Cache of git status

	dirMetaLoaded map[dt.RelDirPath]struct{} // Track when a directory has had its file meta loaded

	// Commit plan persistence (auto-save)
	saveSeq      int           // Auto-save sequence number (incremented on each disposition change)
	saveDebounce time.Duration // Debounce period (default 3 seconds)

	// Alert system for notifications
	Alert bubbleup.AlertModel // Alert overlay for save notifications

	// UI state
	ViewMode ViewMode
	Width    int
	Height   int
}

type EditorStateArgs struct {
	UserRepo *gitutils.Repo
	Logger   *slog.Logger
	Writer   cliutil.Writer
}

func NewEditorState(moduleDir dt.DirPath, args EditorStateArgs) EditorState {
	// Initialize auto-save debounce (3 seconds)
	return EditorState{
		Logger:        args.Logger,
		ModuleDir:     moduleDir,
		UserRepo:      args.UserRepo,
		ViewMode:      FileDispositionsView, // Start in File Selection View
		dirMetaLoaded: make(map[dt.RelDirPath]struct{}),
		Alert: bubbleup.NewAlertModel(50, false, 3*time.Second).
			WithMinWidth(10).
			WithUnicodePrefix().
			WithPosition(bubbleup.TopRightPosition).
			WithAllowEscToClose(),
	}
}

// Ensure EditorState implements tea.Model interface
var _ tea.Model = (*EditorState)(nil)

func (es EditorState) Context() context.Context {
	if es.context == nil {
		es.context = context.Background()
	}
	return es.context
}

// Init implements tea.Model - kicks off file loading and commit plan loading
func (es EditorState) Init() (cmd tea.Cmd) {
	es.Logger.Info("Initializing editor state")
	return tea.Batch(
		bootstrapCmd(es.Logger),
		es.Alert.Init(),
	)
}

// Update implements tea.Model - dispatches to view-specific update handlers
//
//goland:noinspection GoAssignmentToReceiver
func (es EditorState) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	// Delegate all messages to Alert first (for ticks and ESC)
	var alertCmd tea.Cmd
	es, alertCmd = es.alertCmd(msg)

	es.Logger.Info("EditorState.Update()", teaMsgAttrs(msg))

	switch msg := msg.(type) {
	case tea.KeyMsg:

		// Global keys (always work)
		switch msg.String() {
		case "q", "ctrl+c":
			cmd = tea.Quit
		case "ctrl+s":
			// Manual save (Ctrl+S or Cmd+S)
			es.saveSeq++
			cmd = requestSaveCommitPlanCmd(es, es.saveSeq, true)

		case "ctrl+r":
			// Refresh caches (Ctrl+R)
			cmd = es.Init()
		}

	case onErrorMsg:
		cmd = es.Alert.NewAlertCmd(bubbleup.ErrorKey,
			fmt.Sprintf("ERROR: %s, %v", msg.msg, msg.err),
		)

	case tea.WindowSizeMsg:
		// Update EditorState dimensions
		es.Width = msg.Width
		es.Height = msg.Height
		cmd = screenDimensionsCmd(es.Width, es.Height)

	case bootstrapMsg:
		es, cmd = es.ensureModulePath()
		es.layout.Logger = msg.logger
		if cmd == nil {
			cmd = tea.Batch(
				requestChangedFilesCmd(),
				requestGitStatusCmd(),
				requestCommitPlanCmd(),
			)
		}

	case loadChangedFilesMsg:
		cmd = loadChangedFilesCmd(es.Context(), es.UserRepo, es.modulePath)

	case changedFilesLoadedMsg:
		// Files have been loaded, we can now create the layout, load git status, and
		// load commit plan if one already exists.
		es.fileSource = msg.fileSource
		cmd = requestLayoutCmd()

	case loadGitStatusMsg:
		cmd = loadGitStatusCmd(es.UserRepo)

	case loadCommitPlanMsg:
		cmd = loadCommitPlanCmd(es.UserRepo.Root)

	case scheduleSaveCommitPlanMsg:
		es.saveSeq++
		cmd = requestSaveCommitPlanCmd(es, es.saveSeq, false)

	case commitPlanLoadedMsg:
		// Commit plan loaded successfully - update model state
		es.commitPlan = msg.plan
		// Now create the layout
		cmd = requestCommitPlanSetCmd(msg.plan)

	case saveCommitPlanMsg:
		// Save the commit plan
		switch {
		case msg.sequence != es.saveSeq:
			// Debounce timer fired - ignore stale messages
		default:
			err := es.commitPlan.Save(es.UserRepo.Root)
			if err != nil {
				return es, onErrorCmd(
					fmt.Sprintf("Saving commit plan to %s", msg.modRelPath),
					err,
				)
			}
			if msg.showAlert {
				cmd = es.Alert.NewAlertCmd(bubbleup.InfoKey, "Commit plan saved")
			}
		}

	case createLayoutMsg:
		switch {
		case !es.layout.HasDimensions():
			// No dimensions yet; deley, then try again
			cmd = requestLayoutCmd()
		default:
			// Will be called after both WindowSizeMsg and changedFilesLoadedMsg
			cliutil.Stderrf("Must load filesource first.")
			//os.Exit(1)
			es.layout = es.CreateLayout()
			cmd = tea.Batch(
				es.layout.Init(), // Initialize layout (triggers initial directory load)
				resizeLayoutCmd(),
			)
		}

	case directoryMetaLoadedMsg:

		// Enrich files with git status
		for _, file := range msg.childFiles {
			es.GitStatusCache.EnsureFileStatus(file.Path)
			es.setGitStatus(file)
		}

		// Create directory table
		es.dirMetaLoaded[msg.relDirPath] = struct{}{}

		cmd = requestReloadTableCmd(msg.relDirPath, msg.childFiles)

	case gitStatusLoadedMsg:
		// Store in cache
		es.GitStatusCache = msg.statusMap

	case loadDirectoryViewMsg:
		// Defensive check - GitStatusCache should already be loaded from ensureModulePathMsg
		// This should never happen, but if it does, it's an internal error (not user error)
		if es.GitStatusCache == nil {
			cmd = onInternalErrorCmd(
				"GitStatusCache not initialized when requesting directory view",
				fmt.Errorf("directory: %s", msg.relDirPath),
			)
			goto end
		}

		// Double-check if already loaded (second line of defense, in case of race)
		if _, loaded := es.dirMetaLoaded[msg.relDirPath]; loaded {
			// Already loaded, skip
			goto end
		}

		// Trigger directory metadata load (I/O happens in command)
		cmd = loadDirectoryMetaCmd(es.UserRepo.Root, msg.relDirPath, msg.childFiles)

	default:
		// Just here to stop the linter from complaining
	}

end:

	cmds := []tea.Cmd{cmd}

	// Dispatch based on current view mode
	switch es.ViewMode {
	case FileDispositionsView:
		var layoutMsg tea.Msg
		// Delegate to FileDispositionModel for all messages
		layoutMsg, cmd = es.layout.Update(msg)
		cmds = appendCmd(cmds, cmd)
		es.layout = layoutMsg.(FileDispositionModel)
	case TakesView:
		// TODO: Implement takes view update
		cmd = nil
	case FilesView:
		// TODO: Implement files view update
		cmd = nil
	}
	cmds = appendCmd(cmds, alertCmd)

	return es, tea.Batch(cmds...)
}

// View implements tea.Model - dispatches to view-specific renderers
func (es EditorState) View() string {
	// Dispatch based on current view mode
	switch es.ViewMode {
	case FileDispositionsView:
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
	return NewFileDispositionModel(FileDispositionModelArgs{
		Logger:     es.Logger,
		FileSource: es.fileSource,
		UserRepo:   es.UserRepo,
		ModulePath: es.modulePath,
		CommitPlan: es.commitPlan,
		Width:      es.Width,
		Height:     es.Height,
		RepoScoped: es.layout.RepoScoped,
	})
}

// updateFileDispositions handles updates for file selection view
// Returns updated model and command (following BubbleTea's immutable Elm architecture)
//
//goland:noinspection GoAssignmentToReceiver
func (es EditorState) alertCmd(msg tea.Msg) (_ EditorState, cmd tea.Cmd) {
	alertModel, alertCmd := es.Alert.Update(msg)
	es.Alert = alertModel.(bubbleup.AlertModel)
	return es, alertCmd
}

// renderFileSelectionView renders the file selection view
func (es EditorState) renderFileSelectionView() string {
	// Check if layout is initialized before delegating
	if !es.layout.Ready() {
		return "Initializing..."
	}

	// Delegate to FileDispositionModel for rendering
	view := es.layout.View()

	// Overlay alert on top of all content (MUST be last)
	return es.Alert.Render(view)
}

// ensureModulePath ensures that modulePath will have a valid value
func (es EditorState) ensureModulePath() (_ EditorState, cmd tea.Cmd) {
	var err error
	if es.ModuleDir != es.UserRepo.Root {
		// Calculate relative path from repo root to module
		es.modulePath, err = es.ModuleDir.Rel(es.UserRepo.Root)
	}
	if err != nil {
		msg := fmt.Sprintf("Directory mismatch: Module (%s) vs Repo (%s); %v",
			es.ModuleDir,
			es.UserRepo.Root,
			err,
		)
		cmd = onInternalErrorCmd(msg, err)
	}
	return es, cmd
}

// setGitStatus enriches bubbletree.File with git status information
func (es EditorState) setGitStatus(f *bubbletree.File) {
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
	status, found = es.GitStatusCache[f.Path]
	if !found {
		fileData.FileStatus = gitutils.FileStatus{}
		goto end
	}
	fileData.FileStatus = status
	f.SetData(fileData)
end:
	return
}
