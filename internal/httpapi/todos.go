package httpapi

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"tinypanel-hub/internal/domain"
)

const maxTodoTextRunes = 50

func (s *Server) handleGetTodos(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Todos())
}

func (s *Server) handlePostTodo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text   string `json:"text"`
		Status int    `json:"status"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	text, ok := normalizeTodoText(w, req.Text)
	if !ok {
		return
	}
	if !validTodoStatus(req.Status) {
		writeError(w, http.StatusBadRequest, "status must be 0, 1, or 2")
		return
	}

	todo, err := s.store.AddTodo(text, req.Status)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, todo)
}

func (s *Server) handleGetTodo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	todo, found := s.store.Todo(id)
	if !found {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	writeJSON(w, http.StatusOK, todo)
}

func (s *Server) handlePatchTodo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	var req struct {
		Version int64   `json:"version"`
		Text    *string `json:"text"`
		Status  *int    `json:"status"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Version <= 0 {
		writeError(w, http.StatusBadRequest, "version must be a positive integer")
		return
	}
	if req.Text == nil && req.Status == nil {
		writeError(w, http.StatusBadRequest, "text or status is required")
		return
	}

	var patch domain.TodoPatch
	if req.Text != nil {
		text, ok := normalizeTodoText(w, *req.Text)
		if !ok {
			return
		}
		patch.Text = &text
	}
	if req.Status != nil {
		if !validTodoStatus(*req.Status) {
			writeError(w, http.StatusBadRequest, "status must be 0, 1, or 2")
			return
		}
		patch.Status = req.Status
	}

	todo, found, swapped, err := s.store.UpdateTodo(id, req.Version, patch)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	if !swapped {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":           "todo version conflict",
			"current_version": todo.Version,
		})
		return
	}
	writeJSON(w, http.StatusOK, todo)
}

func (s *Server) handleDeleteTodo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	var req struct {
		Version int64 `json:"version"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Version <= 0 {
		writeError(w, http.StatusBadRequest, "version must be a positive integer")
		return
	}

	found, swapped, err := s.store.DeleteTodo(id, req.Version)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	if !swapped {
		writeError(w, http.StatusConflict, "todo version conflict")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func normalizeTodoText(w http.ResponseWriter, text string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return "", false
	}
	if utf8.RuneCountInString(text) > maxTodoTextRunes {
		writeError(w, http.StatusBadRequest, "text must not exceed 50 characters")
		return "", false
	}
	return text, true
}

func validTodoStatus(status int) bool {
	return status >= 0 && status <= 2
}
