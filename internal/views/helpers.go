package views

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"

	"fresnel/internal/domain"
	"fresnel/internal/service"

	"github.com/google/uuid"
)

// Suppress unused import warnings for packages used only in .templ files.
var _ = strings.ToLower

func Lower(v any) string {
	return strings.ToLower(fmt.Sprintf("%v", v))
}

func FmtTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2 Jan 2006 15:04 UTC")
}

func FmtTimestamp(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2006-01-02T15:04:05Z")
}

func FmtDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func FmtUser(id uuid.UUID) string {
	s := id.String()
	if len(s) > 8 {
		return s[:8] + "…"
	}
	return s
}

func FmtBytes(b int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func FmtJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func attachmentSeverityClass(status domain.ScanStatus) string {
	switch status {
	case domain.ScanClean:
		return "info"
	case domain.ScanQuarantined:
		return "high"
	default:
		return "medium"
	}
}

func safeEventField(e *domain.Event, fn func(*domain.Event) string) string {
	if e == nil {
		return ""
	}
	return fn(e)
}

func safeEventDateField(e *domain.Event) string {
	if e == nil || e.OriginalEventDate == nil {
		return ""
	}
	return e.OriginalEventDate.Format("2006-01-02")
}

func EventTypeName(et domain.EventType) string {
	switch et {
	case domain.EventTypePhishing:
		return "Phishing"
	case domain.EventTypeMalware:
		return "Malware"
	case domain.EventTypeRansomware:
		return "Ransomware"
	case domain.EventTypeDDoS:
		return "DDoS"
	case domain.EventTypeDataBreach:
		return "Data Breach"
	case domain.EventTypeUnauthorized:
		return "Unauthorized Access"
	case domain.EventTypeWebDefacement:
		return "Web Defacement"
	case domain.EventTypeInsiderThreat:
		return "Insider Threat"
	case domain.EventTypeSupplyChain:
		return "Supply Chain"
	case domain.EventTypeVulnerability:
		return "Vulnerability"
	case domain.EventTypeHybrid:
		return "Hybrid"
	case domain.EventTypeMisinformation:
		return "Misinformation"
	case domain.EventTypeUnclassified:
		return "Unclassified"
	default:
		return string(et)
	}
}

func safeReportField(r *domain.StatusReport, fn func(*domain.StatusReport) string) string {
	if r == nil {
		return ""
	}
	return fn(r)
}

func safeCampaignField(c *domain.Campaign, fn func(*domain.Campaign) string) string {
	if c == nil {
		return ""
	}
	return fn(c)
}

func dashboardGraphJSON(events []*domain.Event, edges *service.GraphEdges) string {
	type node struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Impact    string `json:"impact"`
		Status    string `json:"status"`
		EventType string `json:"event_type"`
		TLP       string `json:"tlp"`
		UpdatedAt string `json:"updated_at"`
	}
	type edge struct {
		ID        string `json:"id"`
		Source    string `json:"source"`
		Target    string `json:"target"`
		Label     string `json:"label"`
		LineStyle string `json:"line_style"`
	}
	nodes := make([]node, 0, len(events))
	for _, e := range events {
		title := e.Title
		if len(title) > 60 {
			title = title[:57] + "..."
		}
		nodes = append(nodes, node{
			ID:        e.ID.String(),
			Title:     title,
			Impact:    string(e.Impact),
			Status:    string(e.Status),
			EventType: string(e.EventType),
			TLP:       string(e.TLP),
			UpdatedAt: e.UpdatedAt.Format("2006-01-02"),
		})
	}
	edgeList := make([]edge, 0)
	if edges != nil {
		for _, c := range edges.Correlations {
			edgeList = append(edgeList, edge{
				ID:        c.ID.String(),
				Source:    c.EventAID.String(),
				Target:    c.EventBID.String(),
				Label:     c.Label,
				LineStyle: "solid",
			})
		}
		for _, r := range edges.Relationships {
			edgeList = append(edgeList, edge{
				ID:        r.ID.String(),
				Source:    r.SourceEventID.String(),
				Target:    r.TargetEventID.String(),
				Label:     r.Label,
				LineStyle: "dotted",
			})
		}
	}
	b, _ := json.Marshal(map[string]any{"nodes": nodes, "edges": edgeList})
	return string(b)
}

type tlGroup struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Order   int    `json:"order"`
}

func syncTimelineJSON(events []*domain.Event, sectors []*service.DashboardNode) string {
	type item struct {
		ID      string `json:"id"`
		Group   string `json:"group"`
		Content string `json:"content"`
		Start   string `json:"start"`
		End     string `json:"end,omitempty"`
		Type    string `json:"type"`
		Class   string `json:"className"`
		Title   string `json:"title"`
	}

	groupMap := make(map[string]bool)
	var groups []tlGroup
	order := 0
	for _, sec := range sectors {
		collectOrgGroups(sec, &groups, groupMap, &order)
	}

	items := make([]item, 0, len(events))
	for _, e := range events {
		start := e.CreatedAt
		if e.OriginalEventDate != nil && !e.OriginalEventDate.IsZero() {
			start = *e.OriginalEventDate
		}
		var endStr string
		if e.Status == domain.StatusResolved || e.Status == domain.StatusClosed {
			endStr = e.UpdatedAt.Format(time.RFC3339)
		} else {
			endStr = time.Now().UTC().Format(time.RFC3339)
		}

		title := e.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}

		cssClass := "timeline-item-" + strings.ToLower(string(e.Impact))

		orgID := e.OrganizationID.String()
		if !groupMap[orgID] {
			groupMap[orgID] = true
			groups = append(groups, tlGroup{ID: orgID, Content: "Org " + orgID[:8], Order: order})
			order++
		}

		items = append(items, item{
			ID:      e.ID.String(),
			Group:   orgID,
			Content: "<div class='tl-item-inner'><span class='tl-title'>" + template.HTMLEscapeString(title) + "</span>" +
				"<span class='badge badge-impact-" + strings.ToLower(string(e.Impact)) + "' style='font-size:.65rem;padding:.1rem .3rem;'>" + string(e.Impact) + "</span></div>",
			Start:   start.Format(time.RFC3339),
			End:     endStr,
			Type:    "range",
			Class:   cssClass,
			Title:   template.HTMLEscapeString(e.Title) + " (" + string(e.Status) + ")",
		})
	}

	b, _ := json.Marshal(map[string]any{"groups": groups, "items": items})
	return string(b)
}

func collectOrgGroups(node *service.DashboardNode, groups *[]tlGroup, seen map[string]bool, order *int) {
	if node.NodeType == "organization" {
		id := node.ID.String()
		if !seen[id] {
			seen[id] = true
			*groups = append(*groups, tlGroup{ID: id, Content: node.Name, Order: *order})
			*order++
		}
	}
	for _, child := range node.Children {
		collectOrgGroups(child, groups, seen, order)
	}
}

func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	default:
		return FmtDate(t)
	}
}
