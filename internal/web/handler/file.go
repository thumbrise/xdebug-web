package handler

import (
	"io/fs"
	"net/http"
	"os"
	"strings"
)

func File(root string) http.HandlerFunc {
	rootFS := os.DirFS(root)

	return func(w http.ResponseWriter, r *http.Request) {
		relPath := r.URL.Query().Get("path")

		if relPath == "" {
			http.Error(w, "path query parameter is required", http.StatusBadRequest)

			return
		}

		relPath = strings.TrimPrefix(relPath, "/")

		data, err := fs.ReadFile(rootFS, relPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		//nolint:gosec // source code served as text/plain for code editor
		_, _ = w.Write(data)
	}
}
