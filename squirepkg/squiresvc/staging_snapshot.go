package squiresvc

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/squire/squirepkg/squire"
)

// StagingSnapshot represents a snapshot of the staging area
// Used for undo functionality and safety net
type StagingSnapshot struct {
	ID        dt.Identifier  `json:"id"`        // UUID
	Timestamp time.Time      `json:"timestamp"` // When snapshot was taken
	Label     string         `json:"label"`     // Optional user label
	Files     []SnapshotFile `json:"files"`     // Staged files with hunks
	Hash      string         `json:"hash"`      // SHA256 of snapshot content
}

type SnapshotStore struct {
	cfgstore.ConfigStore
	ModuleDir dt.DirPath
}

const StagingSnapshotsPath = "snapshots"

// SnapshotFile represents a file in a staging snapshot
type SnapshotFile struct {
	Path      dt.RelFilepath `json:"path"`       // Relative path from repo root
	StagedAll bool           `json:"staged_all"` // True if entire file is staged
	Hunks     []HunkHeader   `json:"hunks"`      // Specific hunks if not StagedAll
}

// NewSnapshotStore instantiates a new snapshot store object
func NewSnapshotStore(modDir dt.DirPath, ssID dt.Identifier) *SnapshotStore {
	return &SnapshotStore{
		ModuleDir: modDir,
		ConfigStore: cfgstore.NewConfigStore(cfgstore.CustomConfigDirType, cfgstore.ConfigStoreArgs{
			ConfigSlug:  squire.ConfigSlug,
			RelFilepath: dt.RelFilepathJoin(StagingSnapshotsPath, ssID+".json"),
			DirsProvider: cfgstore.DefaultDirsProviderWithArgs(cfgstore.DirsProviderArgs{
				CustomDirPath: modDir,
			}),
		}),
	}
}

// NewStagingSnapshot instantiates a new snapshot object
func NewStagingSnapshot(label string) (ss *StagingSnapshot) {
	// TODO: This will need to call git diff --cached --unified=0
	// and parse the output to extract hunks
	// For now, create basic structure

	ss = &StagingSnapshot{
		ID:        generateSnapshotID(),
		Timestamp: time.Now(),
		Label:     label,
		Files:     make([]SnapshotFile, 0),
	}

	// Compute hash
	ss.Hash = computeSnapshotHash(ss)

	return ss
}

// CreateStagingSnapshot creates a snapshot of the current staging area
// Calls git diff --cached to get staged hunks
func CreateStagingSnapshot(moduleDir dt.DirPath, label string) (ss *StagingSnapshot, err error) {
	// TODO: This will need to call git diff --cached --unified=0
	// and parse the output to extract hunks
	// For now, create basic structure

	ss = NewStagingSnapshot(label)
	store := NewSnapshotStore(moduleDir, ss.ID)
	// Save snapshot
	err = store.Save(ss)

	return ss, err
}

func (store SnapshotStore) Save(ss *StagingSnapshot) (err error) {
	err = store.SaveJSON(ss)
	if err != nil {
		err = fmt.Errorf("failed to write snapshot file: %w", err)
		goto end
	}
end:
	return err
}

// LoadStagingSnapshot loads a snapshot from .squire/snapshots/{id}.json
func LoadStagingSnapshot(modDir dt.DirPath, id dt.Identifier) (ss *StagingSnapshot, store *SnapshotStore, err error) {
	store = NewSnapshotStore(modDir, id)
	ss = &StagingSnapshot{}
	err = store.LoadJSON(ss)
	if err != nil {
		goto end
	}
end:
	return ss, store, err
}

// ListActive lists all active (non-archived) snapshots
func (store SnapshotStore) ListActive() (snapshots []*StagingSnapshot, err error) {
	var entries []os.DirEntry
	var snapshot *StagingSnapshot

	// Read directory
	entries, err = store.ModuleDir.ReadDir()
	if err != nil {
		if os.IsNotExist(err) {
			// No snapshots directory yet - return empty list
			err = nil
			goto end
		}
		err = fmt.Errorf("failed to read snapshots directory: %w", err)
		goto end
	}

	snapshots = make([]*StagingSnapshot, 0, len(entries))

	// Load each snapshot
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .json files
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		name := entry.Name()

		// Extract ID from filename
		id := dt.Identifier(name[:len(name)-5]) // Remove .json

		snapshot, _, err = LoadStagingSnapshot(store.ModuleDir, id)
		if err != nil {
			// Skip invalid snapshots, continue loading others
			continue
		}

		snapshots = append(snapshots, snapshot)
	}

end:
	return snapshots, err
}

// Restore restores the staging area to a previous snapshot
// Applies the snapshot's hunks to git staging
func (store SnapshotStore) Restore() (err error) {

	// TODO: Apply ss to git staging
	// This will require:
	// 1. git reset (clear current staging)
	// 2. git apply for each hunk in ss
	// 3. Or use git add --patch with the stored hunks

	err = fmt.Errorf("RestoreSnapshot not yet implemented")

	//end:
	return err
}

// RestoreStagingSnapshot restores the staging area to a previous snapshot
// Applies the snapshot's hunks to git staging
func RestoreStagingSnapshot(modDir dt.DirPath, id dt.Identifier) error {
	return NewSnapshotStore(modDir, id).Restore()
}

// Archive moves snapshots older than daysOld to .squire/.archive/snapshots/
func (store SnapshotStore) Archive(daysOld int) (archivedCount int, err error) {
	var snapshots []*StagingSnapshot
	var cutoffTime time.Time
	var archiveDir dt.DirPath
	var storeDir dt.DirPath
	var errs []error

	storeDir, err = store.ConfigDir()
	if err != nil {
		goto end
	}

	// Get all active snapshots
	snapshots, err = store.ListActive()
	if err != nil {
		goto end
	}

	cutoffTime = time.Now().AddDate(0, 0, -daysOld)
	archiveDir = dt.DirPathJoin3(storeDir.Dir(), ArchivePath, storeDir.Base())

	// Create archive directory
	err = archiveDir.MkdirAll(0755)
	if err != nil {
		err = fmt.Errorf("failed to create archive snapshots directory: %w", err)
		goto end
	}

	// Archive old snapshots
	for _, snapshot := range snapshots {
		var activeFile dt.Filepath

		if !snapshot.Timestamp.Before(cutoffTime) {
			continue
		}

		// Move to archive
		store.SetRelFilepath(dt.RelFilepath(snapshot.ID + ".json"))

		activeFile, err = store.GetFilepath()
		if err != nil {
			errs = append(errs, err)
			// Log error but continue with other snapshots
			continue
		}
		err = activeFile.Rename(dt.FilepathJoin(archiveDir, activeFile.Base()))
		if err != nil {
			errs = append(errs, err)
			// Log error but continue with other snapshots
			continue
		}

		archivedCount++
	}

	err = CombineErrs(errs)
end:
	return archivedCount, err

}

// ArchiveSnapshots moves snapshots older than daysOld to .squire/.archive/snapshots/
func ArchiveSnapshots(modDir dt.DirPath, daysOld int) (n int, err error) {
	return NewSnapshotStore(modDir, "").Archive(daysOld)
}

// Delete deletes a snapshot (from either active or archive)
func (store SnapshotStore) Delete() (err error) {
	var activeFile dt.Filepath
	var activeDir dt.DirPath
	var archiveFile dt.Filepath
	var errs []error

	activeFile, err = store.GetFilepath()
	if err != nil {
		goto end
	}
	err = activeFile.Remove()
	switch {
	case cfgstore.NoFileOrDirErr(err):
		err = nil
	case err != nil:
		errs = append(errs, NewErr(dt.ErrFailedToRemoveFile, activeFile.ErrKV(), err))
	default:
		// Here for the linter
	}
	activeDir = activeFile.Dir()
	// Try to remove from both locations
	archiveFile = dt.FilepathJoin4(activeDir.Dir(), ArchivePath, activeDir.Base(), activeFile.Base())
	err = archiveFile.Remove()
	switch {
	case cfgstore.NoFileOrDirErr(err):
		err = nil
	case err != nil:
		errs = append(errs, NewErr(dt.ErrFailedToRemoveFile, archiveFile.ErrKV(), err))
	default:
		// Here for the linter
	}
end:
	return CombineErrs(errs)
}

// DeleteSnapshot deletes a snapshot (from either active or archive)
func DeleteSnapshot(modDir dt.DirPath, id dt.Identifier) (err error) {
	return NewSnapshotStore(modDir, id).Delete()
}

// computeSnapshotHash computes a hash of the snapshot content
func computeSnapshotHash(snapshot *StagingSnapshot) string {
	h := sha256.New()

	for _, file := range snapshot.Files {
		h.Write([]byte(file.Path))
		h.Write([]byte("\n"))

		for _, hunk := range file.Hunks {
			h.Write([]byte(hunk.Header))
			h.Write([]byte("\n"))
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

// generateSnapshotID generates a unique ID for a snapshot
// Uses timestamp + random component
func generateSnapshotID() dt.Identifier {
	return dt.Identifier(fmt.Sprintf("snap-%d", time.Now().Unix()))
}
