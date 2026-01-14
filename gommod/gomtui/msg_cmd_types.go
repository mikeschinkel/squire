package gomtui

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/bubbletree"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
	"github.com/mikeschinkel/gomion/gommod/gompkg"
)

// changeDispositionMsg is emitted when a file disposition changes
type changeDispositionMsg struct {
	Node        *FileDispositionNode // Optional: if set, cascade to all descendants (directory)
	Path        dt.RelFilepath       // Single path (used when Node is nil, for files)
	Disposition gompkg.FileDisposition
}

func requestDispositionChangeCmd(item any, fd gompkg.FileDisposition) tea.Cmd {
	return func() tea.Msg {
		msg := changeDispositionMsg{
			Disposition: fd,
		}
		switch t := item.(type) {
		case *FileDispositionNode:
			msg.Node = t
		case dt.RelFilepath:
			msg.Path = t
		}
		return msg
	}
}

type loadChangedFilesMsg struct{}

func requestChangedFilesCmd() tea.Cmd {
	return func() tea.Msg {
		return loadChangedFilesMsg{}
	}
}

type requestGitStatusMsg struct{}

func requestGitStatusCmd() tea.Cmd {
	return func() tea.Msg {
		return loadGitStatusMsg{}
	}
}

type loadCommitPlanMsg struct{}

func requestCommitPlanCmd() tea.Cmd {
	return func() tea.Msg {
		return loadCommitPlanMsg{}
	}
}

func loadChangedFilesCmd(ctx context.Context, repo *gitutils.Repo, modPath dt.RelDirPath) (cmd tea.Cmd) {
	return func() (msg tea.Msg) {
		files, err := LoadChangedRepoFiles(ctx, repo, modPath)
		switch {
		case err != nil:
			msg = onErrorMsg{
				msg: "Loading files",
				err: err,
			}
		default:
			msg = changedFilesLoadedMsg{
				fileSource: NewFileSource(modPath, files),
				repoRoot:   repo.Root,
			}
		}
		return msg
	}
}

type requestFileDispositionUIMsg struct{}

func requestFileDispositionUICmd() tea.Cmd {
	return func() tea.Msg {
		return requestFileDispositionUIMsg{}
	}
}

type changedFilesLoadedMsg struct {
	fileSource *FileSource
	repoRoot   dt.DirPath
}

// commitPlanLoadedMsg - Commit plan loaded successfully
type commitPlanLoadedMsg struct {
	plan *gompkg.CommitPlan
}

// loadCommitPlanCmd performs I/O to load commit plan
func loadCommitPlanCmd(repoRoot dt.DirPath) tea.Cmd {
	return func() tea.Msg {
		plan, err := gompkg.LoadCommitPlan(repoRoot)
		if err != nil {
			return onErrorMsg{
				msg: fmt.Sprintf("Loading commit plan from %s", repoRoot),
				err: err,
			}
		}
		return commitPlanLoadedMsg{
			plan: plan,
		}
	}
}

type saveCommitPlanMsg struct {
	modRelPath dt.RelDirPath
	sequence   int
	showAlert  bool
}

type commitPlanSaveCompleteMsg struct {
	sequence int
}
type resizeLayoutMsg struct{}

func resizeLayoutCmd() tea.Cmd {
	return func() tea.Msg {
		return resizeLayoutMsg{}
	}
}

type scheduleSaveCommitPlanMsg struct{}

type createLayoutMsg struct{}

func requestLayoutCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(100 * time.Millisecond)
		return createLayoutMsg{}
	}
}

func scheduleCommitPlanSaveCmd() tea.Cmd {
	return func() tea.Msg {
		return scheduleSaveCommitPlanMsg{}
	}
}

func requestSaveCommitPlanCmd(es EditorState, seq int, showAlert bool) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(es.saveDebounce)
		return saveCommitPlanMsg{
			modRelPath: es.modulePath,
			sequence:   seq,
			showAlert:  showAlert,
		}
	}
}

type refreshDispositionLayoutMsg struct{}

func refreshDispositionLayoutCmd() tea.Cmd {
	return func() tea.Msg {
		return refreshDispositionLayoutMsg{}
	}
}

// refreshTreeMsg triggers tree row rebuild (async pattern)
type refreshTreeMsg struct{}

func refreshTreeCmd() tea.Cmd {
	return func() tea.Msg {
		return refreshTreeMsg{}
	}
}

// bootstrapMsg boostraps the entire editor state
type bootstrapMsg struct {
	logger *slog.Logger
}

func bootstrapCmd(lgr *slog.Logger) tea.Cmd {
	return func() tea.Msg {
		return bootstrapMsg{logger: lgr}
	}
}

// refreshTableMsg triggers table row rebuild (async pattern)
type setCommitPlanMsg struct {
	plan *gompkg.CommitPlan
}

func requestCommitPlanSetCmd(plan *gompkg.CommitPlan) tea.Cmd {
	return func() tea.Msg {
		return setCommitPlanMsg{
			plan: plan,
		}
	}
}

type refreshTableMsg struct{}

func refreshTableCmd() tea.Cmd {
	return func() tea.Msg {
		return refreshTableMsg{}
	}
}

type screenDimensionsMsg struct {
	Height int
	Width  int
}

func screenDimensionsCmd(w, h int) tea.Cmd {
	return func() tea.Msg {
		return screenDimensionsMsg{
			Width:  w,
			Height: h,
		}
	}
}

// ============================================================================
// Async File Loading Messages (handleLeftPaneFocus refactoring)
// ============================================================================

// loadFileContentMsg - Load file content (does I/O)
type loadFileContentMsg struct {
	path dt.RelFilepath // File path to load
}

func requestFileContentCmd(path dt.RelFilepath) tea.Cmd {
	return func() tea.Msg {
		return loadFileContentMsg{path: path}
	}
}

// fileContentLoadedMsg - File content loaded successfully
type fileContentLoadedMsg struct {
	path    dt.RelFilepath
	content string
	yOffset int
}

// loadFileContentCmd performs I/O to load file content
func loadFileContentCmd(repoRoot dt.DirPath, path dt.RelFilepath) tea.Cmd {
	return func() tea.Msg {
		content, yOffset, err := loadFileContent(repoRoot, path)
		if err != nil {
			return onErrorMsg{
				msg: fmt.Sprintf("Loading content from %s", path),
				err: err,
			}
		}
		return fileContentLoadedMsg{
			path:    path,
			content: content,
			yOffset: yOffset,
		}
	}
}

// loadGitStatusMsg - Load git status (does I/O)
type loadGitStatusMsg struct{}

// gitStatusLoadedMsg - Git status loaded successfully (no error field)
type gitStatusLoadedMsg struct {
	statusMap gitutils.StatusMap
}

func gitStatusLoadedCmd(statusMap gitutils.StatusMap) tea.Cmd {
	return func() tea.Msg {
		return gitStatusLoadedMsg{
			statusMap: statusMap,
		}
	}
}

// loadGitStatusCmd performs I/O to load git status
func loadGitStatusCmd(userRepo *gitutils.Repo) tea.Cmd {
	return func() tea.Msg {
		statusMap, err := loadGitStatus(userRepo)
		if err != nil {
			return onErrorMsg{
				msg: "Loading git status",
				err: err,
			}
		}
		return gitStatusLoadedMsg{
			statusMap: statusMap,
		}
	}
}

// loadDirectoryViewMsg - User selected a directory in the tree, request metadata loaddirectoryMetaLoadedCmd
type loadDirectoryViewMsg struct {
	relDirPath dt.RelDirPath
	childFiles []*bubbletree.File
}

func requestDirectoryViewCmd(dirPath dt.RelDirPath, childFiles []*bubbletree.File) tea.Cmd {
	return func() tea.Msg {
		return loadDirectoryViewMsg{
			relDirPath: dirPath,
			childFiles: childFiles,
		}
	}
}

func loadDirectoryMetaCmd(repoRoot dt.DirPath, dirPath dt.RelDirPath, childFiles []*bubbletree.File) tea.Cmd {
	return func() tea.Msg {
		err := batchLoadMeta(childFiles, repoRoot)
		if err != nil {
			return onErrorCmd(fmt.Sprintf("Loading metadata for directory %s", dirPath), err)
		}
		return directoryMetaLoadedMsg{
			relDirPath: dirPath,
			childFiles: childFiles,
		}
	}
}

// directoryMetaLoadedMsg - Metadata loaded successfully (no error field)
type directoryMetaLoadedMsg struct {
	relDirPath dt.RelDirPath
	childFiles []*bubbletree.File // Files with metadata loaded
}

type reloadTableMsg struct {
	relDirPath dt.RelDirPath
	childFiles []*bubbletree.File // Files with metadata loaded
}

func requestReloadTableCmd(dirPath dt.RelDirPath, childFiles []*bubbletree.File) tea.Cmd {
	return func() tea.Msg {
		return reloadTableMsg{
			relDirPath: dirPath,
			childFiles: childFiles,
		}
	}
}

// ============================================================================
// Background Loading Messages (Sequential, Event-Driven)
// ============================================================================

// loadFileMetaMsg - Load metadata for ONE file, then trigger next
type loadFileMetaMsg struct {
	file           *bubbletree.File
	remainingFiles []*bubbletree.File // Queue of files still to load
}

func loadFileMetaCmd(file *bubbletree.File, remaining []*bubbletree.File) tea.Cmd {
	return func() tea.Msg {
		return loadFileMetaMsg{
			file:           file,
			remainingFiles: remaining,
		}
	}
}

// loadNextFileContentMsg - Load content for ONE file, then trigger next
type loadNextFileContentMsg struct {
	path           dt.RelFilepath
	remainingPaths []dt.RelFilepath // Queue of files still to load
}

func loadNextFileContentCmd(path dt.RelFilepath, remaining []dt.RelFilepath) tea.Cmd {
	return func() tea.Msg {
		return loadNextFileContentMsg{
			path:           path,
			remainingPaths: remaining,
		}
	}
}

// ============================================================================
// Error and Alert Messages
// ============================================================================

type onErrorMsg struct {
	msg      string
	internal bool
	err      error
}

func onErrorCmd(msg string, err error) tea.Cmd {
	return func() tea.Msg {
		return onErrorMsg{
			msg: msg,
			err: err,
		}
	}
}

func onInternalErrorCmd(msg string, err error) tea.Cmd {
	return func() tea.Msg {
		return onErrorMsg{
			msg:      fmt.Sprintf("%s (internal error)", msg),
			internal: true,
			err:      err,
		}
	}
}

type alertMsg struct {
	msg string
}

func alertCmd(msg string) tea.Cmd {
	return func() tea.Msg {
		return alertMsg{
			msg: msg,
		}
	}
}
