package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type FileEntry struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Path     string      `json:"path"`
	Children []FileEntry `json:"children,omitempty"`
}

type FileTreeResponse struct {
	Root    string      `json:"root"`
	Entries []FileEntry `json:"entries"`
}

var ignoreDirs = []string{
	".git", "node_modules", "vendor", ".idea",
	".github", "dist", "build", ".cache",
}

var ignoreFiles = []string{
	".DS_Store", ".gitkeep",
}

func buildFileTree(dirPath string) ([]FileEntry, error) {
	return buildFileTreePrefix(dirPath, "")
}

func buildFileTreePrefix(dirPath, prefix string) ([]FileEntry, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var tree []FileEntry

	for _, entry := range entries {
		name := entry.Name()

		if strings.HasPrefix(name, ".") || slices.Contains(ignoreFiles, name) {
			continue
		}

		relPath := name
		if prefix != "" {
			relPath = prefix + "/" + name
		}

		if entry.IsDir() {
			if slices.Contains(ignoreDirs, name) {
				continue
			}

			children, err := buildFileTreePrefix(filepath.Join(dirPath, name), relPath)
			if err != nil {
				continue
			}

			tree = append(tree, FileEntry{
				Name:     name,
				Type:     "directory",
				Path:     relPath,
				Children: children,
			})
		} else {
			tree = append(tree, FileEntry{
				Name: name,
				Type: "file",
				Path: relPath,
			})
		}
	}

	return tree, nil
}

func Files(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tree, err := buildFileTree(root)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		resp := FileTreeResponse{
			Root:    filepath.Base(root),
			Entries: tree,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
