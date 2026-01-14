package gomtui

// File represents a file with its content for display and its parsed hunks.
// Note: File disposition is stored separately in EditorState.dispositions map.
//type File struct {
//	RelPath    dt.RelFilepath
//	Content string // For display in right pane
//	Hunks   []Hunk
//	YOffset int // Viewport scroll position
//
//	// Cached file metadata (for directory table display)
//	metadata *FileMeta // nil if not yet loaded
//}

//func (f *File) IsEmpty() bool {
//	return f.RelPath == ""
//}

//func (f *File) LoadMeta(root dt.DirPath) (err error) {
//	var filepath dt.Filepath
//	var info os.FileInfo
//
//	// Initialize metadata if not already present
//	if f.metadata != nil {
//		goto end
//	}
//	// Construct full path
//	filepath = dt.FilepathJoin(root, f.RelPath)
//
//	// Get f info using dt.Filepath.Stat() method
//	info, err = filepath.Stat()
//	if err != nil {
//		if os.IsNotExist(err) {
//			// File might be deleted after git status - return nil (not an error)
//			f.metadata = &FileMeta{}
//			err = nil
//			goto end
//		}
//		err = NewErr(ErrFileStatFailed, filepath.ErrKV(), err)
//		goto end
//	}
//
//	// Initialize metadata if not already present
//	f.metadata = &FileMeta{
//		Size:        info.Size(),
//		ModTime:     info.ModTime(),
//		Permissions: info.Mode(),
//		EntryStatus: dt.GetEntryStatus(info),
//	}
//end:
//	return err
//}
