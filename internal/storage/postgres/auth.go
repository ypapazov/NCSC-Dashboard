package postgres

import (
	"context"
	"errors"
	"fmt"

	"fresnel/internal/domain"
	"fresnel/internal/oauth"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotRegistered is returned when no Fresnel user matches the OIDC identity.
var ErrNotRegistered = errors.New("user not registered in Fresnel")

// LoadAuthContext resolves OIDC claims to a domain AuthContext.
func LoadAuthContext(ctx context.Context, pool *pgxpool.Pool, claims *oauth.AccessClaims) (*domain.AuthContext, error) {
	if claims.Sub == "" {
		return nil, fmt.Errorf("missing sub")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var (
		userID                               uuid.UUID
		keycloakSub, displayName, email    string
		primaryOrg                           uuid.UUID
		status                               string
	)
	err = tx.QueryRow(ctx, `
SELECT id, keycloak_sub, display_name, email, primary_org_id, status
FROM fresnel.users
WHERE keycloak_sub = $1 AND status = 'active'`, claims.Sub,
	).Scan(&userID, &keycloakSub, &displayName, &email, &primaryOrg, &status)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		// Link by email on first login
		if claims.Email == "" {
			return nil, ErrNotRegistered
		}
		err = tx.QueryRow(ctx, `
SELECT id, keycloak_sub, display_name, email, primary_org_id, status
FROM fresnel.users
WHERE lower(email) = lower($1) AND status = 'active'`, claims.Email,
		).Scan(&userID, &keycloakSub, &displayName, &email, &primaryOrg, &status)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotRegistered
			}
			return nil, err
		}
		if _, err := tx.Exec(ctx, `UPDATE fresnel.users SET keycloak_sub = $1 WHERE id = $2`, claims.Sub, userID); err != nil {
			return nil, err
		}
		keycloakSub = claims.Sub
	}

	rows, err := tx.Query(ctx, `SELECT organization_id FROM fresnel.user_org_memberships WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	var orgs []uuid.UUID
	for rows.Next() {
		var oid uuid.UUID
		if err := rows.Scan(&oid); err != nil {
			rows.Close()
			return nil, err
		}
		orgs = append(orgs, oid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rrows, err := tx.Query(ctx, `
SELECT role, scope_type, scope_id FROM fresnel_iam.role_assignments WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	var roles []domain.RoleAssignment
	for rrows.Next() {
		var ra domain.RoleAssignment
		if err := rrows.Scan(&ra.Role, &ra.ScopeType, &ra.ScopeID); err != nil {
			rrows.Close()
			return nil, err
		}
		roles = append(roles, ra)
	}
	rrows.Close()
	if err := rrows.Err(); err != nil {
		return nil, err
	}

	brows, err := tx.Query(ctx, `
SELECT scope_type, scope_id FROM fresnel_iam.root_designations WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	var roots []struct {
		t string
		id pgtype.UUID
	}
	for brows.Next() {
		var r struct {
			t string
			id pgtype.UUID
		}
		if err := brows.Scan(&r.t, &r.id); err != nil {
			brows.Close()
			return nil, err
		}
		roots = append(roots, r)
	}
	brows.Close()
	if err := brows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	auth := &domain.AuthContext{
		UserID:           userID,
		KeycloakSub:      keycloakSub,
		DisplayName:      displayName,
		Email:            email,
		PrimaryOrgID:     primaryOrg,
		OrgMemberships:   orgs,
		ActiveOrgContext: primaryOrg,
		Roles:            roles,
		IsRoot:           len(roots) > 0,
		RootScope:        pickRootScope(roots),
	}
	return auth, nil
}

func pickRootScope(roots []struct {
	t  string
	id pgtype.UUID
}) *domain.ScopeEntry {
	priority := []string{"PLATFORM", "SECTOR", "ORG"}
	for _, want := range priority {
		for _, r := range roots {
			if r.t != want {
				continue
			}
			sid := uuid.Nil
			if r.id.Valid {
				sid = uuid.UUID(r.id.Bytes)
			}
			cp := domain.ScopeEntry{Type: r.t, ID: sid}
			return &cp
		}
	}
	return nil
}
