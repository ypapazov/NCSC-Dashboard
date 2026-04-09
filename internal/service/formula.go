package service

import (
	"context"
	"errors"

	"fresnel/internal/domain"
	"fresnel/internal/storage"

	"github.com/google/uuid"
)

// FormulaService wraps persistent formula storage with defaults and validation stubs.
type FormulaService struct {
	store storage.FormulaStore
}

func NewFormulaService(store storage.FormulaStore) *FormulaService {
	return &FormulaService{store: store}
}

// Get returns stored Starlark source for the node, or the default documented formula when none is set.
func (s *FormulaService) Get(ctx context.Context, nodeType string, nodeID *uuid.UUID) (string, error) {
	src, err := s.store.Get(ctx, nodeType, nodeID)
	if err != nil {
		return "", err
	}
	if src == "" {
		return s.GetDefault(), nil
	}
	return src, nil
}

// Set persists a custom formula (not available in this PoC build).
func (s *FormulaService) Set(ctx context.Context, auth *domain.AuthContext, nodeType string, nodeID *uuid.UUID, source string, setBy uuid.UUID) error {
	_ = ctx
	_ = auth
	_ = nodeType
	_ = nodeID
	_ = source
	_ = setBy
	return errors.New("custom formulas are not yet available")
}

// Validate checks Starlark source (not available in this PoC build).
func (s *FormulaService) Validate(source string) error {
	_ = source
	return errors.New("Starlark validation is not yet available")
}

// GetDefault returns the built-in weighted-average formula as a commented documentation string.
func (s *FormulaService) GetDefault() string {
	return `# Fresnel default: weighted average of child assessed status (documentation only).
#
# Intended Starlark shape for dashboard aggregation:
#
# def weighted_average(children):
#     """children: list of structs with .numeric_status (float) and .weight (float, e.g. open event count)."""
#     total_weight = 0.0
#     acc = 0.0
#     for c in children:
#         w = float(c.weight)
#         if w <= 0:
#             continue
#         acc += w * float(c.numeric_status)
#         total_weight += w
#     if total_weight <= 0:
#         return 0.0
#     return acc / total_weight
#
# Map domain.AssessedStatus to numeric_value before calling (NORMAL=0, DEGRADED=1, ...).
`
}
