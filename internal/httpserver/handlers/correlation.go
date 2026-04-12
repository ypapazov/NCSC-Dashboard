package handlers

import (
	"net/http"
	"strings"

	"fresnel/internal/domain"
	"fresnel/internal/httpserver/requestctx"
	"fresnel/internal/service"
	"fresnel/internal/views"

	"github.com/google/uuid"
)

type GraphNode struct {
	ID          uuid.UUID         `json:"id"`
	Title       string            `json:"title"`
	Impact      domain.Impact     `json:"impact"`
	Status      domain.EventStatus `json:"status"`
	EventType   domain.EventType  `json:"event_type"`
	TLP         domain.TLP        `json:"tlp"`
	OrgID       uuid.UUID         `json:"org_id"`
	OrgName     string            `json:"org_name,omitempty"`
	UpdatedAt   string            `json:"updated_at"`
	Shape       string            `json:"shape"`
	BorderColor string            `json:"border_color"`
	FillOpacity float64           `json:"fill_opacity"`
}

type GraphEdge struct {
	ID              string                  `json:"id"`
	Source          uuid.UUID               `json:"source"`
	Target          uuid.UUID               `json:"target"`
	Label           string                  `json:"label"`
	CorrelationType domain.CorrelationType  `json:"correlation_type,omitempty"`
	LineStyle       string                  `json:"line_style"`
	IsRelationship  bool                    `json:"is_relationship"`
}

type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

func eventShape(et domain.EventType) string {
	switch {
	case et == domain.EventTypePhishing || et == domain.EventTypeMalware ||
		et == domain.EventTypeRansomware || et == domain.EventTypeDDoS ||
		et == domain.EventTypeDataBreach || et == domain.EventTypeUnauthorized ||
		et == domain.EventTypeInsiderThreat || et == domain.EventTypeSupplyChain:
		return "ellipse"
	case et == domain.EventTypeVulnerability || et == domain.EventTypeWebDefacement:
		return "diamond"
	case et == domain.EventTypeHybrid || et == domain.EventTypeMisinformation:
		return "hexagon"
	default:
		return "rectangle"
	}
}

func impactColor(i domain.Impact) string {
	switch i {
	case domain.ImpactCritical:
		return "#ef4444"
	case domain.ImpactHigh:
		return "#fb923c"
	case domain.ImpactModerate:
		return "#facc15"
	case domain.ImpactLow:
		return "#3b82f6"
	default:
		return "#6b7280"
	}
}

func statusOpacity(s domain.EventStatus) float64 {
	switch s {
	case domain.StatusOpen, domain.StatusInvestigating:
		return 1.0
	case domain.StatusMitigating:
		return 0.75
	case domain.StatusResolved:
		return 0.50
	case domain.StatusClosed:
		return 0.25
	default:
		return 1.0
	}
}

func corrLineStyle(ct domain.CorrelationType) string {
	switch ct {
	case domain.CorrelationManual:
		return "solid"
	case domain.CorrelationConfirmed:
		return "solid"
	case domain.CorrelationSuggested:
		return "dashed"
	default:
		return "solid"
	}
}

type CorrelationHandler struct {
	corrs   *service.CorrelationService
	events  *service.EventService
	lookups Lookups
}

func NewCorrelationHandler(corrs *service.CorrelationService, events *service.EventService, lk Lookups) *CorrelationHandler {
	return &CorrelationHandler{corrs: corrs, events: events, lookups: lk}
}

func (h *CorrelationHandler) ListByEvent(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	eventID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}

	if r.URL.Query().Get("format") == "graph" {
		h.getGraph(w, r, auth, eventID)
		return
	}

	correlations, err := h.corrs.ListByEvent(r.Context(), auth, eventID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, correlations)
}

func (h *CorrelationHandler) getGraph(w http.ResponseWriter, r *http.Request, auth *domain.AuthContext, seedID uuid.UUID) {
	ctx := r.Context()

	seedEvent, err := h.events.GetByID(ctx, auth, seedID)
	if err != nil {
		respondError(w, r, err)
		return
	}

	nodeMap := make(map[uuid.UUID]*domain.Event)
	nodeMap[seedID] = seedEvent

	corrs, _ := h.corrs.ListByEvent(ctx, auth, seedID)
	rels, _ := h.corrs.ListRelationshipsByEvent(ctx, auth, seedID)

	for _, c := range corrs {
		otherID := c.EventBID
		if otherID == seedID {
			otherID = c.EventAID
		}
		if _, ok := nodeMap[otherID]; !ok {
			if ev, err := h.events.GetByID(ctx, auth, otherID); err == nil && ev != nil {
				nodeMap[otherID] = ev
			}
		}
	}
	for _, rel := range rels {
		for _, rid := range []uuid.UUID{rel.SourceEventID, rel.TargetEventID} {
			if _, ok := nodeMap[rid]; !ok {
				if ev, err := h.events.GetByID(ctx, auth, rid); err == nil && ev != nil {
					nodeMap[rid] = ev
				}
			}
		}
	}

	var nodes []GraphNode
	for _, ev := range nodeMap {
		var orgName string
		if org, _ := h.lookups.Orgs.GetByID(ctx, ev.OrganizationID); org != nil {
			orgName = org.Name
		}
		nodes = append(nodes, GraphNode{
			ID:          ev.ID,
			Title:       ev.Title,
			Impact:      ev.Impact,
			Status:      ev.Status,
			EventType:   ev.EventType,
			TLP:         ev.TLP,
			OrgID:       ev.OrganizationID,
			OrgName:     orgName,
			UpdatedAt:   ev.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			Shape:       eventShape(ev.EventType),
			BorderColor: impactColor(ev.Impact),
			FillOpacity: statusOpacity(ev.Status),
		})
	}

	var edges []GraphEdge
	for _, c := range corrs {
		if _, okA := nodeMap[c.EventAID]; !okA {
			continue
		}
		if _, okB := nodeMap[c.EventBID]; !okB {
			continue
		}
		edges = append(edges, GraphEdge{
			ID:              c.ID.String(),
			Source:          c.EventAID,
			Target:          c.EventBID,
			Label:           c.Label,
			CorrelationType: c.CorrelationType,
			LineStyle:       corrLineStyle(c.CorrelationType),
		})
	}
	for _, rel := range rels {
		if _, okS := nodeMap[rel.SourceEventID]; !okS {
			continue
		}
		if _, okT := nodeMap[rel.TargetEventID]; !okT {
			continue
		}
		edges = append(edges, GraphEdge{
			ID:             rel.ID.String(),
			Source:         rel.SourceEventID,
			Target:         rel.TargetEventID,
			Label:          rel.Label,
			LineStyle:      "dotted",
			IsRelationship: true,
		})
	}

	if getRenderKind(r) != requestctx.RenderJSON {
		respondView(w, r, http.StatusOK, views.GraphPage(seedEvent.Title, seedID.String()))
		return
	}

	respondJSON(w, http.StatusOK, GraphData{Nodes: nodes, Edges: edges})
}

func (h *CorrelationHandler) GraphPage(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	auth := getAuth(r)
	event, err := h.events.GetByID(r.Context(), auth, eventID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = strings.TrimSpace
	respondView(w, r, http.StatusOK, views.GraphPage(event.Title, eventID.String()))
}

func (h *CorrelationHandler) CreateCorrelation(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	eventID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var corr domain.Correlation
	if isFormSubmission(r) {
		_ = r.ParseForm()
		corr.Label = r.FormValue("label")
		corr.CorrelationType = domain.CorrelationType(r.FormValue("correlation_type"))
		if corr.CorrelationType == "" {
			corr.CorrelationType = domain.CorrelationManual
		}
		corr.EventAID = eventID
		if id, err := uuid.Parse(r.FormValue("event_b_id")); err == nil {
			corr.EventBID = id
		}
	} else {
		if err := parseJSON(r, &corr); err != nil {
			respondError(w, r, service.ErrValidation)
			return
		}
	}
	if corr.EventAID == uuid.Nil {
		corr.EventAID = eventID
	}
	if err := h.corrs.CreateCorrelation(r.Context(), auth, &corr); err != nil {
		respondError(w, r, err)
		return
	}
	if getRenderKind(r) != requestctx.RenderJSON {
		w.Header().Set("HX-Redirect", "/events/"+eventID.String())
		w.WriteHeader(http.StatusCreated)
		return
	}
	respondJSON(w, http.StatusCreated, &corr)
}

func (h *CorrelationHandler) ConfirmCorrelation(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.corrs.ConfirmCorrelation(r.Context(), auth, id); err != nil {
		respondError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "confirmed"})
}

func (h *CorrelationHandler) DeleteCorrelation(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.corrs.DeleteCorrelation(r.Context(), auth, id); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CorrelationHandler) CreateRelationship(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	eventID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	var rel domain.EventRelationship
	if err := parseJSON(r, &rel); err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if rel.SourceEventID == uuid.Nil {
		rel.SourceEventID = eventID
	}
	if err := h.corrs.CreateRelationship(r.Context(), auth, &rel); err != nil {
		respondError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, &rel)
}

func (h *CorrelationHandler) ListRelationships(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	eventID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	rels, err := h.corrs.ListRelationshipsByEvent(r.Context(), auth, eventID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, rels)
}
