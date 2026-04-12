package service

import (
	"context"
	"sync"
	"time"

	"fresnel/internal/authz"
	"fresnel/internal/domain"
	"fresnel/internal/storage"

	"github.com/google/uuid"
)

type DashboardNode struct {
	ID             uuid.UUID            `json:"id"`
	Name           string               `json:"name"`
	NodeType       string               `json:"node_type"` // "sector", "organization", or "platform"
	AssessedStatus domain.AssessedStatus `json:"assessed_status"`
	ReportedStatus domain.AssessedStatus `json:"reported_status,omitempty"`
	Children       []*DashboardNode     `json:"children,omitempty"`
	Restricted     bool                 `json:"restricted,omitempty"`
	Depth          int                  `json:"depth"`
	AncestryPath   string               `json:"-"`
}

type DashboardService struct {
	sectors  storage.SectorStore
	orgs     storage.OrganizationStore
	reports  storage.StatusReportStore
	authz    authz.Authorizer

	mu       sync.RWMutex
	cache    *DashboardNode
	cacheAt  time.Time
	cacheTTL time.Duration
}

func NewDashboardService(sectors storage.SectorStore, orgs storage.OrganizationStore, reports storage.StatusReportStore, az authz.Authorizer, cacheTTL time.Duration) *DashboardService {
	if cacheTTL <= 0 {
		cacheTTL = 60 * time.Second
	}
	return &DashboardService{
		sectors: sectors, orgs: orgs, reports: reports,
		authz: az, cacheTTL: cacheTTL,
	}
}

func (s *DashboardService) GetTree(ctx context.Context, auth *domain.AuthContext) (*DashboardNode, error) {
	s.mu.RLock()
	if s.cache != nil && time.Since(s.cacheAt) < s.cacheTTL {
		cached := s.cache
		s.mu.RUnlock()
		return s.filterTree(ctx, auth, cached), nil
	}
	s.mu.RUnlock()

	tree, err := s.buildTree(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cache = tree
	s.cacheAt = time.Now()
	s.mu.Unlock()

	return s.filterTree(ctx, auth, tree), nil
}

func (s *DashboardService) InvalidateCache() {
	s.mu.Lock()
	s.cache = nil
	s.mu.Unlock()
}

func (s *DashboardService) buildTree(ctx context.Context) (*DashboardNode, error) {
	allSectors, err := s.sectors.List(ctx)
	if err != nil {
		return nil, err
	}

	sectorMap := make(map[uuid.UUID]*DashboardNode)
	var topLevel []*DashboardNode

	for _, sec := range allSectors {
		node := &DashboardNode{
			ID:           sec.ID,
			Name:         sec.Name,
			NodeType:     "sector",
			Depth:        sec.Depth,
			AncestryPath: sec.AncestryPath,
		}
		sectorMap[sec.ID] = node
	}

	for _, sec := range allSectors {
		node := sectorMap[sec.ID]
		if sec.ParentSectorID != nil {
			if parent, ok := sectorMap[*sec.ParentSectorID]; ok {
				parent.Children = append(parent.Children, node)
			} else {
				topLevel = append(topLevel, node)
			}
		} else {
			topLevel = append(topLevel, node)
		}
	}

	for _, sec := range allSectors {
		node := sectorMap[sec.ID]
		latest, _ := s.reports.GetLatestByScope(ctx, "SECTOR", sec.ID)
		if latest != nil {
			node.ReportedStatus = latest.AssessedStatus
		}

		orgs, err := s.orgs.List(ctx, &sec.ID)
		if err != nil {
			continue
		}
		for _, org := range orgs {
			orgNode := &DashboardNode{
				ID:       org.ID,
				Name:     org.Name,
				NodeType: "organization",
				Depth:    node.Depth + 1,
			}
			orgLatest, _ := s.reports.GetLatestByScope(ctx, "ORG", org.ID)
			if orgLatest != nil {
				orgNode.AssessedStatus = orgLatest.AssessedStatus
				orgNode.ReportedStatus = orgLatest.AssessedStatus
			} else {
				orgNode.AssessedStatus = domain.AssessedUnknown
			}
			node.Children = append(node.Children, orgNode)
		}
	}

	// Compute parent statuses via weighted average (bottom-up)
	for _, top := range topLevel {
		computeStatus(top)
	}

	root := &DashboardNode{
		Name:     "Platform",
		NodeType: "platform",
		Children: topLevel,
	}
	computeStatus(root)
	return root, nil
}

func computeStatus(node *DashboardNode) domain.AssessedStatus {
	if len(node.Children) == 0 {
		return node.AssessedStatus
	}
	var sum float64
	var count int
	for _, child := range node.Children {
		childStatus := computeStatus(child)
		v := childStatus.NumericValue()
		if v >= 0 {
			sum += v
			count++
		}
	}
	if count == 0 {
		node.AssessedStatus = domain.AssessedUnknown
		return node.AssessedStatus
	}
	avg := sum / float64(count)
	switch {
	case avg < 0.5:
		node.AssessedStatus = domain.AssessedNormal
	case avg < 1.5:
		node.AssessedStatus = domain.AssessedDegraded
	case avg < 2.5:
		node.AssessedStatus = domain.AssessedImpaired
	default:
		node.AssessedStatus = domain.AssessedCritical
	}
	return node.AssessedStatus
}

func (s *DashboardService) filterTree(ctx context.Context, auth *domain.AuthContext, node *DashboardNode) *DashboardNode {
	if node == nil {
		return nil
	}
	copy := *node
	copy.Children = nil

	for _, child := range node.Children {
		res := &authz.Resource{
			Type:           child.NodeType,
			ID:             child.ID,
			SectorAncestry: child.AncestryPath,
		}
		if child.NodeType == "organization" {
			res.Type = "Organization"
			res.OrganizationID = child.ID
		} else {
			res.Type = "Sector"
		}

		if s.authz.Authorize(ctx, auth, authz.ActionView, res) {
			filtered := s.filterTree(ctx, auth, child)
			copy.Children = append(copy.Children, filtered)
		} else {
			restricted := &DashboardNode{
				ID:             child.ID,
				Name:           child.Name,
				NodeType:       child.NodeType,
				AssessedStatus: domain.AssessedUnknown,
				Restricted:     true,
				Depth:          child.Depth,
			}
			copy.Children = append(copy.Children, restricted)
		}
	}
	return &copy
}

type TimelineEntry struct {
	Type      string    `json:"type"` // "event" or "status_report"
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
	Impact    string    `json:"impact"`
	TLP       string    `json:"tlp"`
}
