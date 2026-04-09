package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	KeycloakSub  string    `json:"keycloak_sub"`
	DisplayName  string    `json:"display_name"`
	Email        string    `json:"email"`
	PrimaryOrgID uuid.UUID `json:"primary_org_id"`
	Timezone     string    `json:"timezone"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

func (u *User) Validate() error {
	if u.DisplayName == "" {
		return fmt.Errorf("display_name is required")
	}
	if len(u.DisplayName) > 255 {
		return fmt.Errorf("display_name too long (max 255)")
	}
	if u.Email == "" {
		return fmt.Errorf("email is required")
	}
	if len(u.Email) > 320 {
		return fmt.Errorf("email too long (max 320)")
	}
	if u.PrimaryOrgID == uuid.Nil {
		return fmt.Errorf("primary_org_id is required")
	}
	return nil
}

type UserOrgMembership struct {
	UserID         uuid.UUID `json:"user_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	AssignedBy     uuid.UUID `json:"assigned_by"`
	AssignedAt     time.Time `json:"assigned_at"`
}
