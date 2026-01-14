package gomtui

// ViewMode represents which view the user is currently in
type ViewMode int

const (
	FileDispositionsView ViewMode = iota
	TakesView
	FilesView
)

func (v ViewMode) String() string {
	switch v {
	case FileDispositionsView:
		return "File Selection"
	case TakesView:
		return "Takes"
	case FilesView:
		return "Files"
	default:
		return "Unknown"
	}
}
