package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/domain"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/repository"
	"github.com/zhanglei10281852-gif/campus-agent-lab-go/internal/requestmeta"
)

type userUpdateRequest struct {
	Email       string            `json:"email"`
	DisplayName string            `json:"display_name"`
	Role        domain.Role       `json:"role"`
	Status      domain.UserStatus `json:"status"`
	Password    string            `json:"password"`
	Version     int64             `json:"version"`
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	current, size := queryPage(r)
	page, err := s.services.Auth.ListUsers(r.Context(), repository.UserFilter{
		Page:   repository.PageRequest{Limit: size, Offset: (current - 1) * size},
		Email:  r.URL.Query().Get("username"),
		Status: domain.UserStatus(r.URL.Query().Get("status")),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"records": page.Items, "total": page.Total, "current": current, "size": size})
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	var input userUpdateRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	if input.Version < 1 {
		writeError(w, r, domain.FieldError{Field: "version", Message: "must be positive"})
		return
	}
	user, err := s.services.Auth.UpdateUser(r.Context(), parseID(r), input.Version, input.DisplayName, input.Role, input.Status, input.Password)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) updateUserStatus(w http.ResponseWriter, r *http.Request) {
	statusValue, err := strconv.Atoi(r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, r, domain.FieldError{Field: "status", Message: "must be 0 or 1"})
		return
	}
	status := domain.UserActive
	if statusValue == 0 {
		status = domain.UserDisabled
	} else if statusValue != 1 {
		writeError(w, r, domain.FieldError{Field: "status", Message: "must be 0 or 1"})
		return
	}
	user, err := s.services.Auth.GetUser(r.Context(), parseID(r))
	if err != nil {
		writeError(w, r, err)
		return
	}
	updated, err := s.services.Auth.UpdateUser(r.Context(), parseID(r), user.Version, user.DisplayName, user.Role, status, "")
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	version, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil {
		writeError(w, r, domain.FieldError{Field: "version", Message: "must be an integer"})
		return
	}
	if err := s.services.Auth.DeleteUser(r.Context(), parseID(r), version); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type profileUpdateRequest struct {
	DisplayName string `json:"display_name"`
}
type passwordUpdateRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	var input profileUpdateRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	principal, _ := requestmeta.Principal(r.Context())
	updated, err := s.services.Auth.UpdateProfile(r.Context(), principal, strings.TrimSpace(input.DisplayName))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) updatePassword(w http.ResponseWriter, r *http.Request) {
	var input passwordUpdateRequest
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, err)
		return
	}
	principal, _ := requestmeta.Principal(r.Context())
	if err := s.services.Auth.UpdatePassword(r.Context(), principal, input.OldPassword, input.NewPassword); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func queryPage(r *http.Request) (int, int) {
	current, _ := strconv.Atoi(r.URL.Query().Get("current"))
	if current < 1 {
		current = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	return current, size
}
