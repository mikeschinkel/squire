package gompkg

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gomion"
)

// CommitCandidate represents an AI-generated commit message
// Tied to a specific staging state via hash
type CommitCandidate struct {
	ID           dt.Identifier `json:"id"`            // UUID
	Message      string        `json:"message"`       // The commit message
	StagingHash  string        `json:"staging_hash"`  // SHA256 of staged files/hunks
	AnalysisHash string        `json:"analysis_hash"` // Hash of analysis results used
	Created      time.Time     `json:"created"`       // Creation timestamp
	Modified     time.Time     `json:"modified"`      // Last modification timestamp
	AIProvider   string        `json:"ai_provider"`   // e.g., "anthropic", "openai"
	AIModel      string        `json:"ai_model"`      // e.g., "claude-sonnet-4", "gpt-4"
	PlanID       dt.Identifier `json:"plan_id"`       // Associated staging plan ID (optional)
	Archived     bool          `json:"archived"`      // True if archived (stale)
}

type CandidateStore struct {
	cfgstore.ConfigStore
	ModuleDir   dt.DirPath
	CandidateId dt.Identifier
}

const CommitCandidatesPath = "candidates"

// NewCandidateStore instantiates a new candidate store object
func NewCandidateStore(modDir dt.DirPath, ccId dt.Identifier) *CandidateStore {
	return &CandidateStore{
		ModuleDir:   modDir,
		CandidateId: ccId,
		ConfigStore: cfgstore.NewConfigStore(cfgstore.CustomConfigDirType, cfgstore.ConfigStoreArgs{
			ConfigSlug:  gomion.ConfigSlug,
			RelFilepath: dt.RelFilepathJoin(CommitCandidatesPath, ccId+".json"),
			DirsProvider: cfgstore.DefaultDirsProviderWithArgs(cfgstore.DirsProviderArgs{
				CustomDirPath: modDir,
			}),
		}),
	}
}

// NewCommitCandidate instantiates a new commit candidate object
func NewCommitCandidate(message string) (cc *CommitCandidate) {
	now := time.Now()
	cc = &CommitCandidate{
		ID:       generateCandidateID(),
		Message:  message,
		Created:  now,
		Modified: now,
	}
	return cc
}

// Save saves the commit candidate to .gomion/candidates/{id}.json or archive
func (store CandidateStore) Save(cc *CommitCandidate) (err error) {
	// Update Modified timestamp
	cc.Modified = time.Now()

	// If archiving, handle moving to archive
	if cc.Archived {
		var archiveFile dt.Filepath

		// Save to archive location
		fp := store.GetRelFilepath()
		dp := fp.Dir()
		ccId := cc.ID + ".json"
		archiveFile = dt.FilepathJoin4(dp.Dir(), ArchivePath, dp.Base(), ccId)

		// Create archive directory
		err = archiveFile.Dir().MkdirAll(0755)
		if err != nil {
			err = fmt.Errorf("failed to create archive candidates directory: %w", err)
			goto end
		}

		// Save to archive
		err = archiveFile.WriteFile([]byte{}, 0644)
		if err != nil {
			goto end
		}
		store.SetRelFilepath(dt.RelFilepathJoin3(ArchivePath, CommitCandidatesPath, ccId))
	}

	err = store.SaveJSON(cc)
	if err != nil {
		err = fmt.Errorf("failed to write candidate file: %w", err)
		goto end
	}

	// If archived, remove from active directory
	if cc.Archived {
		var activeFile dt.Filepath
		fp := store.GetRelFilepath()
		dp := fp.Dir()
		ccId := cc.ID + ".json"
		activeFile = dt.FilepathJoin(dp, ccId)
		err = activeFile.Remove()
		if cfgstore.NoFileOrDirErr(err) {
			err = nil
		}
	}

end:
	return err
}

// LoadCommitCandidate loads a commit candidate from .gomion/candidates/{id}.json
func LoadCommitCandidate(modDir dt.DirPath, id dt.Identifier) (cc *CommitCandidate, store *CandidateStore, err error) {
	store = NewCandidateStore(modDir, id)
	cc = &CommitCandidate{}
	err = store.LoadJSON(cc)
	if err != nil {
		// Try archive if not found in active
		if os.IsNotExist(err) {
			store.SetRelFilepath(dt.RelFilepathJoin3(ArchivePath, CommitCandidatesPath, id+".json"))
			err = store.LoadJSON(cc)
		}
		if err != nil {
			goto end
		}
	}
end:
	return cc, store, err
}

// ListActive lists all active (non-archived) candidates
func (store CandidateStore) ListActive() (candidates []*CommitCandidate, err error) {
	var entries []os.DirEntry
	var candidate *CommitCandidate

	// Get config directory
	configDir, err := store.ConfigDir()
	if err != nil {
		goto end
	}

	// Read directory
	entries, err = configDir.ReadDir()
	if err != nil {
		if os.IsNotExist(err) {
			// No candidates directory yet - return empty list
			err = nil
			goto end
		}
		err = fmt.Errorf("failed to read candidates directory: %w", err)
		goto end
	}

	candidates = make([]*CommitCandidate, 0, len(entries))

	// Load each candidate
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .json files
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Extract ID from filename
		id := dt.Identifier(entry.Name()[:len(entry.Name())-5]) // Remove .json

		candidate, _, err = LoadCommitCandidate(store.ModuleDir, id)
		if err != nil {
			// Skip invalid candidates, continue loading others
			continue
		}

		// Only include non-archived candidates
		if !candidate.Archived {
			candidates = append(candidates, candidate)
		}
	}

end:
	return candidates, err
}

// ListActiveCandidates lists all active (non-archived) commit candidates
func ListActiveCandidates(modDir dt.DirPath) (candidates []*CommitCandidate, err error) {
	return NewCandidateStore(modDir, "").ListActive()
}

// Archive marks a candidate as archived and moves it to .gomion/.archive/candidates/
func (store CandidateStore) Archive() (err error) {
	var cc *CommitCandidate

	err = store.LoadJSON(&cc)
	if err != nil {
		goto end
	}
	// Mark as archived
	cc.Archived = true

	// Save (will move to archive directory)
	err = store.Save(cc)
	// TODO Shouldn't we also delete the active one?
end:
	return err
}

// ArchiveCandidate marks a candidate as archived and moves it to .gomion/.archive/candidates/
func ArchiveCandidate(modDir dt.DirPath, id dt.Identifier) (err error) {
	return NewCandidateStore(modDir, id).Archive()
}

// Delete deletes a candidate (from either active or archive)
func (store CandidateStore) Delete() (err error) {
	var activeFile dt.Filepath
	var activeDir dt.DirPath
	var archiveFile dt.Filepath
	var errs []error

	activeFile, err = store.GetFilepath()
	if err != nil {
		goto end
	}
	err = activeFile.Remove()
	if cfgstore.NoFileOrDirErr(err) {
		err = nil
	}
	if err != nil {
		errs = append(errs, NewErr(dt.ErrFailedToRemoveFile, activeFile.ErrKV(), err))
	}
	activeDir = activeFile.Dir()
	// Try to remove from archive location
	archiveFile = dt.FilepathJoin4(activeDir.Dir(), ArchivePath, activeDir.Base(), store.CandidateId)
	err = archiveFile.Remove()
	if cfgstore.NoFileOrDirErr(err) {
		err = nil
	}
	if err != nil {
		errs = append(errs, NewErr(dt.ErrFailedToRemoveFile, archiveFile.ErrKV(), err))
	}
	err = CombineErrs(errs)
end:
	return err
}

// DeleteCommitCandidate deletes a commit candidate (from either active or archive)
func DeleteCommitCandidate(modDir dt.DirPath, id dt.Identifier) (err error) {
	return NewCandidateStore(modDir, id).Delete()
}

// ComputeStagingHash computes a hash of the current staging area
// Used to detect when staging has changed since a candidate was generated
func ComputeStagingHash(stagedFiles []dt.RelFilepath) (hash string) {
	h := sha256.New()

	// Hash all staged file paths in sorted order
	for _, file := range stagedFiles {
		h.Write([]byte(file))
		h.Write([]byte("\n"))
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

// IsCandidateStale checks if a candidate is stale (staging has changed)
func IsCandidateStale(candidate *CommitCandidate, currentStagingHash string) bool {
	return candidate.StagingHash != currentStagingHash
}

// generateCandidateID generates a unique ID for a commit candidate
// Uses timestamp + random component
func generateCandidateID() dt.Identifier {
	return dt.Identifier(fmt.Sprintf("cand-%d", time.Now().Unix()))
}
