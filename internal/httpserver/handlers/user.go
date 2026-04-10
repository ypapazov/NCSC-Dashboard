package handlers

import (
	"net/http"

	"fresnel/internal/domain"
	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/service"
	"fresnel/internal/views"

	"github.com/google/uuid"
)

type UserHandler struct {
	users *service.UserService
}

func NewUserHandler(users *service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	q := r.URL.Query()
	p := parsePagination(r)

	var orgID *uuid.UUID
	if v := q.Get("organization_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			orgID = &id
		}
	}

	result, err := h.users.List(r.Context(), auth, orgID, p)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, UserListData{
			User:  auth,
			Users: result.Items,
			Total: result.TotalCount,
		})
		return
	}
	respondView(w, r, http.StatusOK, views.AdminUsers(result.Items, result.TotalCount))
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	user, err := h.users.GetByID(r.Context(), auth, id)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, UserDetailData{
			User:        auth,
			ProfileUser: user,
		})
		return
	}
	respondView(w, r, http.StatusOK, views.UserDetail(user))
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	user, err := h.users.GetMe(r.Context(), auth)
	if err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, UserDetailData{
			User:        auth,
			ProfileUser: user,
		})
		return
	}
	respondView(w, r, http.StatusOK, views.UserDetail(user))
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	var user domain.User
	if err := parseJSON(r, &user); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.users.Create(r.Context(), auth, &user); err != nil {
		respondError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, &user)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var user domain.User
	if err := parseJSON(r, &user); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	user.ID = id
	if err := h.users.Update(r.Context(), auth, &user); err != nil {
		respondError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, &user)
}

type roleRequest struct {
	Role      domain.Role      `json:"role"`
	ScopeType domain.ScopeType `json:"scope_type"`
	ScopeID   uuid.UUID        `json:"scope_id"`
}

func (h *UserHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	userID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var req roleRequest
	if err := parseJSON(r, &req); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.users.AssignRole(r.Context(), auth, userID, req.Role, req.ScopeType, req.ScopeID); err != nil {
		respondError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]string{"status": "role_assigned"})
}

func (h *UserHandler) RevokeRole(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	userID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var req roleRequest
	if err := parseJSON(r, &req); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.users.RevokeRole(r.Context(), auth, userID, req.Role, req.ScopeType, req.ScopeID); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) Form(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	var user *domain.User

	if idStr := r.PathValue("id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
		user, err = h.users.GetByID(r.Context(), auth, id)
		if err != nil {
			respondError(w, r, err)
			return
		}
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, UserFormData{User: auth, ProfileUser: user})
		return
	}
	respondView(w, r, http.StatusOK, views.UserForm(user))
}
