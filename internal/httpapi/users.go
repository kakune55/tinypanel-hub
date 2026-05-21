package httpapi

import (
	"net/http"
	"strings"
)

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		APIToken string `json:"api_token"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.APIToken = strings.TrimSpace(req.APIToken)
	if req.APIToken == "" {
		req.APIToken = randomSecret()
	}

	user, err := s.services.Users.Create(req.Name, req.Email, tokenHash(req.APIToken))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"user":      publicUser(user),
		"api_token": req.APIToken,
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	writeJSON(w, http.StatusOK, publicUser(user))
}
