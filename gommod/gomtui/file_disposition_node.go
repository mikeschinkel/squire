package gomtui

import (
	"github.com/mikeschinkel/go-dt"
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

// GetAllDescendantPaths recursively collects all file paths under a node (including the node itself)
// Used for cascading disposition changes to all files in a directory
func GetAllDescendantPaths(node *FileDispositionNode) []dt.RelFilepath {
	if node == nil {
		return nil
	}

	var paths []dt.RelFilepath

	// Add this node's path
	data := node.Data()
	if data != nil {
		paths = append(paths, data.Path)
	}

	// Recursively add all children's paths
	if node.HasChildren() {
		for _, child := range node.Children() {
			childPaths := GetAllDescendantPaths(child)
			paths = append(paths, childPaths...)
		}
	}

	return paths
}
