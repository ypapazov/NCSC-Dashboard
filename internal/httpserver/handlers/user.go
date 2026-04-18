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
	users   *service.UserService
	orgs    *service.OrganizationService
	lookups Lookups
}

func NewUserHandler(users *service.UserService, orgs *service.OrganizationService, lk Lookups) *UserHandler {
	return &UserHandler{users: users, orgs: orgs, lookups: lk}
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
	orgNames := make(views.NameMap)
	for _, u := range result.Items {
		if _, ok := orgNames[u.PrimaryOrgID]; !ok {
			if o, err := h.orgs.GetByID(r.Context(), auth, u.PrimaryOrgID); err == nil && o != nil {
				orgNames[u.PrimaryOrgID] = o.Name
			}
		}
	}
	respondView(w, r, http.StatusOK, views.AdminUsers(result.Items, result.TotalCount, orgNames))
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
	orgName := user.PrimaryOrgID.String()
	if o, err := h.orgs.GetByID(r.Context(), auth, user.PrimaryOrgID); err == nil && o != nil {
		orgName = o.Name
	}
	respondView(w, r, http.StatusOK, views.UserDetail(user, orgName))
}

func (h *UserHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
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
	ctx := r.Context()

	roles, _ := h.lookups.Roles.ListRoles(ctx, id)
	var roleViews []views.RoleView
	for _, ra := range roles {
		name := ra.ScopeID.String()
		if ra.ScopeType == string(domain.ScopePlatform) {
			name = "Platform"
		} else if ra.ScopeType == string(domain.ScopeSector) {
			if s, err := h.lookups.Sectors.GetByID(ctx, ra.ScopeID); err == nil && s != nil {
				name = s.Name
			}
		} else if ra.ScopeType == string(domain.ScopeOrg) {
			if o, err := h.lookups.Orgs.GetByID(ctx, ra.ScopeID); err == nil && o != nil {
				name = o.Name
			}
		}
		roleViews = append(roleViews, views.RoleView{
			Role: ra.Role, ScopeType: ra.ScopeType, ScopeName: name, ScopeID: ra.ScopeID,
		})
	}

	var scopeOptions []views.ScopeOption
	if sectors, _ := h.lookups.Sectors.List(ctx); sectors != nil {
		for _, s := range sectors {
			scopeOptions = append(scopeOptions, views.ScopeOption{ID: s.ID, Name: s.Name, Type: "SECTOR"})
		}
	}
	if orgs, _ := h.lookups.Orgs.List(ctx, nil); orgs != nil {
		for _, o := range orgs {
			scopeOptions = append(scopeOptions, views.ScopeOption{ID: o.ID, Name: o.Name, Type: "ORG"})
		}
	}

	respondView(w, r, http.StatusOK, views.AdminRoles(views.AdminRolesData{
		UserID:       id,
		UserName:     user.DisplayName,
		Roles:        roleViews,
		ScopeOptions: scopeOptions,
	}))
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
	orgName := user.PrimaryOrgID.String()
	if o, err := h.orgs.GetByID(r.Context(), auth, user.PrimaryOrgID); err == nil && o != nil {
		orgName = o.Name
	}
	respondView(w, r, http.StatusOK, views.UserDetail(user, orgName))
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	var user *domain.User
	var password string
	if isFormSubmission(r) {
		user = parseUserFromForm(r)
		password = r.FormValue("password")
	} else {
		user = &domain.User{}
		if err := parseJSON(r, user); err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
	}
	if err := h.users.Create(r.Context(), auth, user, password); err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusCreated, user)
		return
	}
	w.Header().Set("HX-Redirect", "/users")
	w.WriteHeader(http.StatusCreated)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var user *domain.User
	if isFormSubmission(r) {
		user = parseUserFromForm(r)
	} else {
		user = &domain.User{}
		if err := parseJSON(r, user); err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
	}
	user.ID = id
	if err := h.users.Update(r.Context(), auth, user); err != nil {
		respondError(w, r, err)
		return
	}

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, user)
		return
	}
	w.Header().Set("HX-Redirect", "/users")
	w.WriteHeader(http.StatusOK)
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
	if isFormSubmission(r) {
		_ = r.ParseForm()
		req.Role = domain.Role(r.FormValue("role"))
		req.ScopeType = domain.ScopeType(r.FormValue("scope_type"))
		if sid := r.FormValue("scope_id"); sid != "" {
			req.ScopeID, _ = uuid.Parse(sid)
		}
	} else if err := parseJSON(r, &req); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.users.AssignRole(r.Context(), auth, userID, req.Role, req.ScopeType, req.ScopeID); err != nil {
		respondError(w, r, err)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/users/"+userID.String())
		w.WriteHeader(http.StatusCreated)
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
	if q := r.URL.Query(); q.Get("role") != "" {
		req.Role = domain.Role(q.Get("role"))
		req.ScopeType = domain.ScopeType(q.Get("scope_type"))
		if sid := q.Get("scope_id"); sid != "" {
			req.ScopeID, _ = uuid.Parse(sid)
		}
	} else if err := parseJSON(r, &req); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.users.RevokeRole(r.Context(), auth, userID, req.Role, req.ScopeType, req.ScopeID); err != nil {
		respondError(w, r, err)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/users/"+userID.String())
		w.WriteHeader(http.StatusOK)
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

	orgs, _ := h.orgs.List(r.Context(), auth, nil)

	if getRenderKind(r) == requestctx.RenderJSON {
		respondJSON(w, http.StatusOK, UserFormData{User: auth, ProfileUser: user})
		return
	}
	respondView(w, r, http.StatusOK, views.UserForm(user, orgs))
}
