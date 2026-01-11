package gomtui

import (
	"github.com/mikeschinkel/gomion/gommod/bubbletree"
)

type FileDispositionNode = bubbletree.FileNode

// getChildFiles extracts File pointers from a tree node's direct children.
// Filters out folders (nodes with children) and returns only files.
// Used to get the list of files to display in directory table.

type NodeChildrenGetter[T any] interface {
	Children() []*bubbletree.Node[T]
}

func GetNodeChildData[T any](getter NodeChildrenGetter[T]) (datum []*T) {
	if getter == nil {
		return nil
	}
	children := getter.Children()
	datum = make([]*T, 0, len(children))
	for _, child := range children {
		// Skip folders - only include leaf nodes (files)
		if !child.HasChildren() {
			data := child.Data()
			if data != nil {
				datum = append(datum, data)
			}
		}
	}
	return datum
}
