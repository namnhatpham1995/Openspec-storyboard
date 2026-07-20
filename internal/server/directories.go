package server

import (
	"errors"
	"io/fs"
	"net/http"

	"storyboard/internal/directorybrowser"
)

type directoryLister interface {
	List(path string) (directorybrowser.Listing, error)
}

func (s *Server) handleDirectories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	listing, err := s.directories.List(r.URL.Query().Get("path"))
	if err == nil {
		writeJSON(w, http.StatusOK, listing)
		return
	}

	switch {
	case errors.Is(err, directorybrowser.ErrPathNotAbsolute):
		writeAPIError(w, http.StatusBadRequest, "invalid_directory_path", "directory path must be absolute")
	case errors.Is(err, directorybrowser.ErrNotDirectory):
		writeAPIError(w, http.StatusUnprocessableEntity, "not_a_directory", "path must identify a directory")
	case errors.Is(err, fs.ErrNotExist):
		writeAPIError(w, http.StatusNotFound, "directory_not_found", "directory does not exist")
	case errors.Is(err, fs.ErrPermission):
		writeAPIError(w, http.StatusForbidden, "directory_inaccessible", "directory is not accessible")
	default:
		s.logger.ErrorContext(r.Context(), "listing directory", "path", r.URL.Query().Get("path"), "error", err)
		writeAPIError(w, http.StatusInternalServerError, "directory_read_failed", "could not read the directory")
	}
}
