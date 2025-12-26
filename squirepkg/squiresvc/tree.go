package squiresvc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// TreeNode represents a node in the dependency tree
type TreeNode struct {
	Module   *Module
	Children []*TreeNode
}

// TreeOptions controls how the tree is rendered
type TreeOptions struct {
	ShowDirs  bool
	ShowPaths bool
	ShowAll   bool
	ShowExt   bool
}

// RenderTree renders a dependency tree starting from root modules
func (ms *ModuleSet) RenderTree(opts TreeOptions) (output string, err error) {
	var rootModules []*Module
	var module *Module
	var lines []string
	var node *TreeNode

	// Find root modules (modules in the current repo)
	rootModules = ms.findRootModules()

	if len(rootModules) == 0 {
		output = "No modules found\n"
		goto end
	}

	lines = make([]string, 0)

	// Build and render tree for each root module
	for _, module = range rootModules {
		node = ms.buildTreeNode(module, make(map[ModulePath]struct{}))
		ms.renderNode(node, "", true, &lines, opts)
	}

	output = strings.Join(lines, "\n")
	if output != "" {
		output += "\n"
	}

end:
	return output, err
}

// findRootModules finds modules in the current repo
func (ms *ModuleSet) findRootModules() (roots []*Module) {
	var module *Module
	var firstRepoRoot dt.DirPath

	roots = make([]*Module, 0)

	if len(ms.Modules) == 0 {
		goto end
	}

	// Get the first module's repo root as reference
	firstRepoRoot = ms.Modules[0].RepoRoot

	// Find all modules in the same repo
	for _, module = range ms.Modules {
		if module.RepoRoot == firstRepoRoot {
			roots = append(roots, module)
		}
	}

end:
	return roots
}

// buildTreeNode builds a tree node for a module and its dependencies
func (ms *ModuleSet) buildTreeNode(module *Module, visited map[ModulePath]struct{}) (node *TreeNode) {
	var depPath ModulePath
	var depModule *Module
	var ok bool
	var childNode *TreeNode

	node = &TreeNode{
		Module:   module,
		Children: make([]*TreeNode, 0),
	}

	// Don't expand if we've already visited this module
	if _, ok := visited[module.ModulePath]; !ok {
		goto end
	}

	visited[module.ModulePath] = struct{}{}

	// Build children for each dependency
	for _, depPath = range module.Requires {
		depModule, ok = ms.Get(depPath)
		if !ok {
			continue
		}

		childNode = ms.buildTreeNode(depModule, visited)
		node.Children = append(node.Children, childNode)
	}

end:
	return node
}

// renderNode renders a tree node and its children
func (ms *ModuleSet) renderNode(node *TreeNode, prefix string, isLast bool, lines *[]string, opts TreeOptions) {
	var line string
	var label string
	var childPrefix string
	var i int
	var child *TreeNode
	var isLastChild bool

	// Build the label for this node
	label = ms.buildNodeLabel(node.Module, opts)

	// Build the tree branch characters
	if prefix == "" {
		// Root level - no prefix
		line = label
	} else {
		if isLast {
			line = prefix + "└─ " + label
		} else {
			line = prefix + "├─ " + label
		}
	}

	*lines = append(*lines, line)

	// Build prefix for children
	if isLast {
		childPrefix = prefix + "   "
	} else {
		childPrefix = prefix + "│  "
	}

	// Render children
	for i, child = range node.Children {
		isLastChild = i == len(node.Children)-1
		ms.renderNode(child, childPrefix, isLastChild, lines, opts)
	}
}

// buildNodeLabel builds the label for a tree node based on options
func (ms *ModuleSet) buildNodeLabel(module *Module, opts TreeOptions) (label string) {
	var homeDir dt.DirPath
	var err error
	var modulePath ModulePath
	var location string

	if opts.ShowDirs {
		// Show relative directory for local modules
		label = string(module.RelDir)
		goto end
	}

	if opts.ShowAll {
		// Show both module path and location
		modulePath = module.ModulePath

		// Try to get home directory for tilde expansion
		homeDir, err = dt.UserHomeDir()
		if err == nil {
			// Convert to path relative to home with tilde
			location = ms.pathWithTilde(module.RepoRoot, homeDir)
		} else {
			location = string(module.RepoRoot)
		}

		label = fmt.Sprintf("%s (~%s)", modulePath, location)
		goto end
	}

	if opts.ShowPaths {
		// Show full module path
		label = string(module.ModulePath)
		goto end
	}

	// Default: show short name (human-readable, no stutter)
	label = module.ShortName()

end:
	return label
}

// pathWithTilde converts an absolute path to use ~ for home directory
func (ms *ModuleSet) pathWithTilde(path dt.DirPath, homeDir dt.DirPath) (result string) {
	var absPath string
	var homePath string
	var relPath string
	var err error

	absPath = string(path)
	homePath = string(homeDir)

	// Check if path is under home directory
	relPath, err = filepath.Rel(homePath, absPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		// Not under home directory
		result = absPath
		goto end
	}

	// Convert to tilde path
	if relPath == "." {
		result = "~"
	} else {
		result = filepath.Join("~", relPath)
	}

end:
	return result
}

// EmbedTree embeds a rendered tree into a markdown file
func EmbedTree(markdownPath dt.Filepath, treeContent string, before bool) (err error) {
	var content []byte
	var lines []string
	var i int
	var line string
	var markerIndex int
	var newLines []string
	var codeBlock string
	var newContent string

	// Read the markdown file
	content, err = markdownPath.ReadFile()
	if err != nil {
		goto end
	}

	// Split into lines
	lines = strings.Split(string(content), "\n")

	// Find the marker
	markerIndex = -1
	for i, line = range lines {
		if strings.TrimSpace(line) == "<!-- squire:embed-requires-tree -->" {
			markerIndex = i
			break
		}
	}

	if markerIndex == -1 {
		err = fmt.Errorf("marker <!-- squire:embed-requires-tree --> not found in %s", markdownPath)
		goto end
	}

	// Build the code block
	codeBlock = "```text\n" + treeContent + "```"

	// Build new content
	newLines = make([]string, 0, len(lines)+4)

	if before {
		// Insert before marker
		newLines = append(newLines, lines[:markerIndex]...)
		newLines = append(newLines, codeBlock)
		newLines = append(newLines, lines[markerIndex:]...)
	} else {
		// Insert after marker
		newLines = append(newLines, lines[:markerIndex+1]...)
		newLines = append(newLines, codeBlock)
		if markerIndex+1 < len(lines) {
			newLines = append(newLines, lines[markerIndex+1:]...)
		}
	}

	newContent = strings.Join(newLines, "\n")

	// Write back to file
	err = os.WriteFile(string(markdownPath), []byte(newContent), 0o644)
	if err != nil {
		goto end
	}

end:
	return err
}
