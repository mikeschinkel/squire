package gomtui

import (
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
)

type FileData struct {
	FileStatus gitutils.FileStatus
	Hunks      []Hunk
	YOffset    int // Viewport scroll position
}

func (fd FileData) Load(fp dt.Filepath) (err error) {
	panic("not implemented")
	return err
}

func NewFileData() *FileData {
	return &FileData{
		FileStatus: gitutils.FileStatus{},
		Hunks:      make([]Hunk, 0),
		YOffset:    0,
	}
}
