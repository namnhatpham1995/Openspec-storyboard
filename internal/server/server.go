package server

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"storyboard/internal/openspec"
)

// Server serves the read-only Storyboard API for one project. The registry
// phase will replace this single-project stepping stone with dynamic projects.
type Server struct {
	projectRoot string
	projectFS   fs.FS
	logger      *slog.Logger
}

// New constructs a read-only API server rooted at projectRoot.
func New(projectRoot string, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		projectRoot: projectRoot,
		projectFS:   os.DirFS(projectRoot),
		logger:      logger,
	}
}

// NewWithFS is like New, but accepts an fs.FS for focused tests.
func NewWithFS(projectRoot string, projectFS fs.FS, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{projectRoot: projectRoot, projectFS: projectFS, logger: logger}
}

// Handler returns the complete HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/projects/current", s.handleCurrentProject)
	mux.HandleFunc("GET /api/changes/{name}", s.handleChangeDetail)
	return s.logRequests(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCurrentProject(w http.ResponseWriter, _ *http.Request) {
	project, err := openspec.Discover(s.projectFS, s.projectRoot)
	if err != nil {
		s.writeReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) handleChangeDetail(w http.ResponseWriter, r *http.Request) {
	detail, err := openspec.ReadChangeDetail(s.projectFS, r.PathValue("name"))
	if err != nil {
		s.writeReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) writeReadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, openspec.ErrChangeNotFound):
		writeAPIError(w, http.StatusNotFound, "change_not_found", err.Error())
	case errors.Is(err, openspec.ErrNotOpenSpecProject):
		writeAPIError(w, http.StatusUnprocessableEntity, "not_openspec_project", err.Error())
	default:
		s.logger.Error("reading OpenSpec project", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "read_failed", "could not read the OpenSpec project")
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		s.logger.InfoContext(r.Context(), "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.status,
			"duration", time.Since(started),
		)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}
