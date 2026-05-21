package httpapi

import (
	"net/http"
	"strings"
)

func (s *Server) handleDeviceHello(w http.ResponseWriter, r *http.Request) {
	deviceID := strings.TrimSpace(r.Header.Get("X-Device-ID"))
	if deviceID == "" {
		var req struct {
			DeviceID string `json:"device_id"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		deviceID = strings.TrimSpace(req.DeviceID)
	}
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	rawSecret := strings.TrimSpace(r.Header.Get("X-Device-Secret"))
	newSecret := false
	if rawSecret == "" {
		rawSecret = randomSecret()
		newSecret = true
	}

	hello, err := s.services.Devices.Hello(deviceID, tokenHash(rawSecret))
	if err != nil {
		if err.Error() == "invalid device secret" {
			writeError(w, http.StatusUnauthorized, "missing or invalid device credentials")
			return
		}
		s.writeStoreError(w, err)
		return
	}
	if newSecret {
		hello.DeviceSecret = rawSecret
	}
	writeJSON(w, http.StatusOK, hello)
}

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	writeJSON(w, http.StatusOK, publicDevices(s.services.Devices.List(user.ID)))
}

func (s *Server) handleBindDevice(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	var req struct {
		BindCode string `json:"bind_code"`
		Name     string `json:"name"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.BindCode = strings.TrimSpace(req.BindCode)
	req.Name = strings.TrimSpace(req.Name)
	if req.BindCode == "" {
		writeError(w, http.StatusBadRequest, "bind_code is required")
		return
	}

	device, found, bound, err := s.services.Devices.Bind(user.ID, req.BindCode, req.Name)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "bind code not found")
		return
	}
	if !bound {
		writeError(w, http.StatusConflict, "bind code expired or already used")
		return
	}
	writeJSON(w, http.StatusOK, publicDevice(device))
}

func (s *Server) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	device, ok := s.services.Devices.Get(user.ID, pathString(r, "device_id"))
	if !ok {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	writeJSON(w, http.StatusOK, publicDevice(device))
}

func (s *Server) handlePatchDevice(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	var req struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	device, ok, err := s.services.Devices.Update(user.ID, pathString(r, "device_id"), name)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	writeJSON(w, http.StatusOK, publicDevice(device))
}

func (s *Server) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	ok, err := s.services.Devices.Delete(user.ID, pathString(r, "device_id"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
