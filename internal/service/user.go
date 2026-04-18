package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"fresnel/internal/authz"
	"fresnel/internal/domain"
	"fresnel/internal/keycloak"
	"fresnel/internal/storage"

	"github.com/google/uuid"
)

type UserService struct {
	users storage.UserStore
	roles storage.RoleStore
	authz authz.Authorizer
	audit *AuditService
	kc    *keycloak.AdminClient // nil when KC admin creds not configured
}

func NewUserService(users storage.UserStore, roles storage.RoleStore, az authz.Authorizer, audit *AuditService, kc *keycloak.AdminClient) *UserService {
	return &UserService{users: users, roles: roles, authz: az, audit: audit, kc: kc}
}

func (s *UserService) Create(ctx context.Context, auth *domain.AuthContext, user *domain.User, password string) error {
	res := authz.UserResource(user)
	if !s.authz.Authorize(ctx, auth, authz.ActionCreate, res) {
		return ErrForbidden
	}
	if err := user.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	user.ID = uuid.New()
	user.Status = "active"
	if user.Timezone == "" {
		user.Timezone = "UTC"
	}

	if password != "" && s.kc != nil {
		nameParts := strings.SplitN(user.DisplayName, " ", 2)
		firstName := nameParts[0]
		lastName := ""
		if len(nameParts) > 1 {
			lastName = nameParts[1]
		}
		kcSub, err := s.kc.CreateUser(ctx, user.Email, firstName, lastName, password)
		if err != nil {
			return fmt.Errorf("keycloak provisioning failed: %w", err)
		}
		user.KeycloakSub = kcSub
		slog.Info("keycloak user provisioned", "email", user.Email, "kc_sub", kcSub)
	} else if user.KeycloakSub == "" {
		user.KeycloakSub = "pending-" + user.ID.String()
	}

	if err := s.users.Create(ctx, user); err != nil {
		return err
	}
	if err := s.users.AddOrgMembership(ctx, user.ID, user.PrimaryOrgID, auth.UserID); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "create", "user", &user.ID, domain.SeverityInfo, map[string]any{
		"email": user.Email, "primary_org_id": user.PrimaryOrgID,
	})
	return nil
}

func (s *UserService) GetByID(ctx context.Context, _ *domain.AuthContext, id uuid.UUID) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	return user, nil
}

func (s *UserService) GetMe(ctx context.Context, auth *domain.AuthContext) (*domain.User, error) {
	return s.GetByID(ctx, auth, auth.UserID)
}

func (s *UserService) List(ctx context.Context, auth *domain.AuthContext, orgID *uuid.UUID, p domain.Pagination) (*domain.ListResult[*domain.User], error) {
	p.Normalize()
	return s.users.List(ctx, orgID, p)
}

func (s *UserService) Update(ctx context.Context, auth *domain.AuthContext, user *domain.User) error {
	existing, err := s.users.GetByID(ctx, user.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}
	// Users can update themselves; others need manage_members
	if auth.UserID != user.ID {
		res := authz.UserResource(existing)
		if !s.authz.Authorize(ctx, auth, authz.ActionEdit, res) {
			return ErrForbidden
		}
	}
	existing.DisplayName = user.DisplayName
	existing.Email = user.Email
	if user.Timezone != "" {
		existing.Timezone = user.Timezone
	}
	if err := existing.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := s.users.Update(ctx, existing); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "update", "user", &user.ID, domain.SeverityInfo, map[string]any{"email": existing.Email})
	return nil
}

func (s *UserService) Delete(ctx context.Context, auth *domain.AuthContext, id uuid.UUID) error {
	existing, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}
	res := authz.UserResource(existing)
	if !s.authz.Authorize(ctx, auth, authz.ActionDelete, res) {
		return ErrForbidden
	}
	if err := s.users.Delete(ctx, id); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "delete", "user", &id, domain.SeverityHigh, map[string]any{
		"email": existing.Email,
	})
	return nil
}

func (s *UserService) AssignRole(ctx context.Context, auth *domain.AuthContext, userID uuid.UUID, role domain.Role, scopeType domain.ScopeType, scopeID uuid.UUID) error {
	res := &authz.Resource{Type: "User", ID: userID, OrganizationID: scopeID}
	if !s.authz.Authorize(ctx, auth, authz.ActionManageRoles, res) {
		return ErrForbidden
	}
	if err := s.roles.AssignRole(ctx, userID, role, scopeType, scopeID, auth.UserID); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "assign_role", "user", &userID, domain.SeverityHigh, map[string]any{
		"role": role, "scope_type": scopeType, "scope_id": scopeID,
	})
	return nil
}

func (s *UserService) RevokeRole(ctx context.Context, auth *domain.AuthContext, userID uuid.UUID, role domain.Role, scopeType domain.ScopeType, scopeID uuid.UUID) error {
	res := &authz.Resource{Type: "User", ID: userID, OrganizationID: scopeID}
	if !s.authz.Authorize(ctx, auth, authz.ActionManageRoles, res) {
		return ErrForbidden
	}
	if err := s.roles.RevokeRole(ctx, userID, role, scopeType, scopeID); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "revoke_role", "user", &userID, domain.SeverityHigh, map[string]any{
		"role": role, "scope_type": scopeType, "scope_id": scopeID,
	})
	return nil
}

func (s *UserService) DesignateRoot(ctx context.Context, auth *domain.AuthContext, userID uuid.UUID, scopeType domain.ScopeType, scopeID *uuid.UUID) error {
	res := &authz.Resource{Type: "User", ID: userID}
	if !s.authz.Authorize(ctx, auth, authz.ActionManageRoles, res) {
		return ErrForbidden
	}
	if err := s.roles.DesignateRoot(ctx, userID, scopeType, scopeID, auth.UserID); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "designate_root", "user", &userID, domain.SeverityHigh, map[string]any{
		"scope_type": scopeType, "scope_id": scopeID,
	})
	return nil
}

func (s *UserService) AddOrgMembership(ctx context.Context, auth *domain.AuthContext, userID, orgID uuid.UUID) error {
	res := &authz.Resource{Type: "Organization", ID: orgID, OrganizationID: orgID}
	if !s.authz.Authorize(ctx, auth, authz.ActionManageMembers, res) {
		return ErrForbidden
	}
	if err := s.users.AddOrgMembership(ctx, userID, orgID, auth.UserID); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "add_org_membership", "user", &userID, domain.SeverityInfo, map[string]any{"organization_id": orgID})
	return nil
}

func (s *UserService) RemoveOrgMembership(ctx context.Context, auth *domain.AuthContext, userID, orgID uuid.UUID) error {
	res := &authz.Resource{Type: "Organization", ID: orgID, OrganizationID: orgID}
	if !s.authz.Authorize(ctx, auth, authz.ActionManageMembers, res) {
		return ErrForbidden
	}
	if err := s.users.RemoveOrgMembership(ctx, userID, orgID); err != nil {
		return err
	}
	s.audit.Log(ctx, auth, "remove_org_membership", "user", &userID, domain.SeverityInfo, map[string]any{"organization_id": orgID})
	return nil
}
