package gomtui

// getChildFiles extracts File pointers from a tree node's direct children.
// Filters out folders (nodes with children) and returns only files.
// Used to get the list of files to display in directory table.
func getChildFiles(node *FileDispositionNode) (files []*File) {
	if node == nil {
		return nil
	}

	children := node.Children()
	files = make([]*File, 0, len(children))

	for _, child := range children {
		// Skip folders - only include leaf nodes (files)
		if !child.HasChildren() {
			fileData := child.Data()
			if fileData != nil {
				files = append(files, fileData)
			}
		}
	}

	return files
}
