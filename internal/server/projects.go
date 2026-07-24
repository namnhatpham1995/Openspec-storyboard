package server

import (
	"errors"
	"net/http"
	"os"

	"storyboard/internal/openspec"
	projectregistry "storyboard/internal/registry"
)

type registeredProjectView struct {
	ID         string            `json:"id"`
	Path       string            `json:"path"`
	Name       string            `json:"name"`
	Connected  bool              `json:"connected"`
	SchemaName string            `json:"schemaName,omitempty"`
	Changes    []openspec.Change `json:"changes,omitempty"`
	Error      string            `json:"error,omitempty"`
}

func (s *Server) handleProjects(w http.ResponseWriter, _ *http.Request) {
	projects := s.registry.List()
	views := make([]registeredProjectView, 0, len(projects))
	for _, registered := range projects {
		view := readRegisteredProject(registered)
		if view.Connected {
			s.startProjectWatcher(registered)
		}
		views = append(views, view)
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": views})
}

type addProjectRequest struct {
	Path string `json:"path"`
}

type toggleTaskRequest struct {
	Version openspec.FileVersion `json:"version"`
}

type taskTextRequest struct {
	Text    string               `json:"text"`
	Version openspec.FileVersion `json:"version"`
}

type artifactTextRequest struct {
	Content string               `json:"content"`
	Version openspec.FileVersion `json:"version"`
}

func (s *Server) handleAddProject(w http.ResponseWriter, r *http.Request) {
	var request addProjectRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	project, err := s.registry.Add(request.Path)
	if err != nil {
		if errors.Is(err, projectregistry.ErrInvalidProject) {
			writeAPIError(w, http.StatusUnprocessableEntity, "invalid_project", "folder must contain an openspec directory")
			return
		}
		s.logger.Error("registering project", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "registry_write_failed", "could not save the project registry")
		return
	}
	s.startProjectWatcher(project)
	writeJSON(w, http.StatusCreated, readRegisteredProject(project))
}

func (s *Server) handleRemoveProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("projectID")
	if err := s.registry.Remove(id); err != nil {
		if errors.Is(err, projectregistry.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "project_not_found", err.Error())
			return
		}
		s.logger.Error("removing project", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "registry_write_failed", "could not save the project registry")
		return
	}
	s.stopProjectWatcher(id)
	w.WriteHeader(http.StatusNoContent)
}

func readRegisteredProject(registered projectregistry.Project) registeredProjectView {
	view := registeredProjectView{ID: registered.ID, Path: registered.Path, Name: registered.Name}
	project, err := openspec.Discover(os.DirFS(registered.Path), registered.Path)
	if err != nil {
		view.Error = "Project folder is unavailable or no longer contains openspec/."
		return view
	}
	view.Connected = true
	view.SchemaName = project.SchemaName
	view.Changes = project.Changes
	return view
}

func (s *Server) registeredProject(w http.ResponseWriter, id string) (projectregistry.Project, bool) {
	project, ok := s.registry.Get(id)
	if !ok {
		writeAPIError(w, http.StatusNotFound, "project_not_found", "registered project not found")
	}
	return project, ok
}

func (s *Server) handleRegisteredChangeDetail(w http.ResponseWriter, r *http.Request) {
	project, ok := s.registeredProject(w, r.PathValue("projectID"))
	if !ok {
		return
	}
	detail, err := openspec.ReadChangeDetail(os.DirFS(project.Path), r.PathValue("name"))
	if err != nil {
		s.writeReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleRegisteredTaskToggle(w http.ResponseWriter, r *http.Request) {
	project, ok := s.registeredProject(w, r.PathValue("projectID"))
	if !ok {
		return
	}
	var request toggleTaskRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := openspec.ToggleTaskFile(project.Path, r.PathValue("name"), r.PathValue("id"), request.Version)
	if err != nil {
		s.writeReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleRegisteredTaskText(w http.ResponseWriter, r *http.Request) {
	project, ok := s.registeredProject(w, r.PathValue("projectID"))
	if !ok {
		return
	}
	var request taskTextRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := openspec.UpdateTaskTextFile(project.Path, r.PathValue("name"), r.PathValue("id"), request.Text, request.Version)
	if err != nil {
		s.writeReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleRegisteredArtifactText(w http.ResponseWriter, r *http.Request) {
	project, ok := s.registeredProject(w, r.PathValue("projectID"))
	if !ok {
		return
	}
	var request artifactTextRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := openspec.SaveArtifactFile(project.Path, r.PathValue("name"), r.PathValue("path"), request.Content, request.Version)
	if err != nil {
		s.writeReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
