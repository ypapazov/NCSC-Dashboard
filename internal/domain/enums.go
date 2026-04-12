package domain

import "fmt"

// TLP represents the Traffic Light Protocol marking.
type TLP string

const (
	TLPClear       TLP = "CLEAR"
	TLPGreen       TLP = "GREEN"
	TLPAmber       TLP = "AMBER"
	TLPAmberStrict TLP = "AMBER_STRICT"
	TLPRed         TLP = "RED"
)

func (t TLP) Valid() bool {
	switch t {
	case TLPClear, TLPGreen, TLPAmber, TLPAmberStrict, TLPRed:
		return true
	}
	return false
}

// Restrictiveness returns a numeric level (higher = more restrictive).
func (t TLP) Restrictiveness() int {
	switch t {
	case TLPClear:
		return 0
	case TLPGreen:
		return 1
	case TLPAmber:
		return 2
	case TLPAmberStrict:
		return 3
	case TLPRed:
		return 4
	}
	return -1
}

type Impact string

const (
	ImpactInfo     Impact = "INFO"
	ImpactLow      Impact = "LOW"
	ImpactModerate Impact = "MODERATE"
	ImpactHigh     Impact = "HIGH"
	ImpactCritical Impact = "CRITICAL"
)

func (i Impact) Valid() bool {
	switch i {
	case ImpactInfo, ImpactLow, ImpactModerate, ImpactHigh, ImpactCritical:
		return true
	}
	return false
}

func (i Impact) Severity() int {
	switch i {
	case ImpactInfo:
		return 0
	case ImpactLow:
		return 1
	case ImpactModerate:
		return 2
	case ImpactHigh:
		return 3
	case ImpactCritical:
		return 4
	}
	return -1
}

type EventStatus string

const (
	StatusOpen          EventStatus = "OPEN"
	StatusInvestigating EventStatus = "INVESTIGATING"
	StatusMitigating    EventStatus = "MITIGATING"
	StatusResolved      EventStatus = "RESOLVED"
	StatusClosed        EventStatus = "CLOSED"
)

func (s EventStatus) Valid() bool {
	switch s {
	case StatusOpen, StatusInvestigating, StatusMitigating, StatusResolved, StatusClosed:
		return true
	}
	return false
}

func (s EventStatus) IsOpen() bool {
	return s != StatusResolved && s != StatusClosed
}

var validTransitions = map[EventStatus][]EventStatus{
	StatusOpen:          {StatusInvestigating, StatusMitigating, StatusResolved, StatusClosed},
	StatusInvestigating: {StatusMitigating, StatusResolved, StatusClosed},
	StatusMitigating:    {StatusResolved, StatusClosed},
	StatusResolved:      {StatusClosed},
	StatusClosed:        {},
}

func (s EventStatus) CanTransitionTo(next EventStatus) bool {
	for _, v := range validTransitions[s] {
		if v == next {
			return true
		}
	}
	return false
}

type AssessedStatus string

const (
	AssessedNormal   AssessedStatus = "NORMAL"
	AssessedDegraded AssessedStatus = "DEGRADED"
	AssessedImpaired AssessedStatus = "IMPAIRED"
	AssessedCritical AssessedStatus = "CRITICAL"
	AssessedUnknown  AssessedStatus = "UNKNOWN"
)

func (a AssessedStatus) Valid() bool {
	switch a {
	case AssessedNormal, AssessedDegraded, AssessedImpaired, AssessedCritical, AssessedUnknown:
		return true
	}
	return false
}

func (a AssessedStatus) NumericValue() float64 {
	switch a {
	case AssessedNormal:
		return 0
	case AssessedDegraded:
		return 1
	case AssessedImpaired:
		return 2
	case AssessedCritical:
		return 3
	}
	return -1
}

type EventType string

const (
	EventTypePhishing        EventType = "PHISHING"
	EventTypeMalware         EventType = "MALWARE"
	EventTypeRansomware      EventType = "RANSOMWARE"
	EventTypeDDoS            EventType = "DDOS"
	EventTypeDataBreach      EventType = "DATA_BREACH"
	EventTypeUnauthorized    EventType = "UNAUTHORIZED_ACCESS"
	EventTypeWebDefacement   EventType = "WEB_DEFACEMENT"
	EventTypeInsiderThreat   EventType = "INSIDER_THREAT"
	EventTypeSupplyChain     EventType = "SUPPLY_CHAIN"
	EventTypeVulnerability   EventType = "VULNERABILITY"
	EventTypeHybrid          EventType = "HYBRID"
	EventTypeMisinformation  EventType = "MISINFORMATION"
	EventTypeUnclassified    EventType = "UNCLASSIFIED"
)

func (e EventType) Valid() bool {
	switch e {
	case EventTypePhishing, EventTypeMalware, EventTypeRansomware, EventTypeDDoS,
		EventTypeDataBreach, EventTypeUnauthorized, EventTypeWebDefacement,
		EventTypeInsiderThreat, EventTypeSupplyChain, EventTypeVulnerability,
		EventTypeHybrid, EventTypeMisinformation, EventTypeUnclassified:
		return true
	}
	return false
}

type Role string

const (
	RolePlatformRoot Role = "PLATFORM_ROOT"
	RoleSectorRoot   Role = "SECTOR_ROOT"
	RoleOrgRoot      Role = "ORG_ROOT"
	RoleOrgAdmin     Role = "ORG_ADMIN"
	RoleContentAdmin Role = "CONTENT_ADMIN"
	RoleContributor  Role = "CONTRIBUTOR"
	RoleViewer       Role = "VIEWER"
	RoleLiaison      Role = "LIAISON"
)

func (r Role) Valid() bool {
	switch r {
	case RolePlatformRoot, RoleSectorRoot, RoleOrgRoot, RoleOrgAdmin,
		RoleContentAdmin, RoleContributor, RoleViewer, RoleLiaison:
		return true
	}
	return false
}

type ScopeType string

const (
	ScopePlatform ScopeType = "PLATFORM"
	ScopeSector   ScopeType = "SECTOR"
	ScopeOrg      ScopeType = "ORG"
)

func (s ScopeType) Valid() bool {
	switch s {
	case ScopePlatform, ScopeSector, ScopeOrg:
		return true
	}
	return false
}

type CorrelationType string

const (
	CorrelationManual    CorrelationType = "MANUAL"
	CorrelationSuggested CorrelationType = "SUGGESTED"
	CorrelationConfirmed CorrelationType = "CONFIRMED"
)

type ScanStatus string

const (
	ScanPending     ScanStatus = "pending"
	ScanClean       ScanStatus = "clean"
	ScanQuarantined ScanStatus = "quarantined"
	ScanError       ScanStatus = "error"
)

type CampaignStatus string

const (
	CampaignActive   CampaignStatus = "ACTIVE"
	CampaignClosed   CampaignStatus = "CLOSED"
)

type AuditSeverity string

const (
	SeverityInfo   AuditSeverity = "INFO"
	SeverityMedium AuditSeverity = "MEDIUM"
	SeverityHigh   AuditSeverity = "HIGH"
)

type Pagination struct {
	Offset int
	Limit  int
}

func (p *Pagination) Normalize() {
	if p.Offset < 0 {
		p.Offset = 0
	}
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 25
	}
}

type ListResult[T any] struct {
	Items      []T
	TotalCount int
}

// ValidateEnum is a helper for validating string enums.
func ValidateEnum(name, value string, valid func() bool) error {
	if !valid() {
		return fmt.Errorf("invalid %s: %q", name, value)
	}
	return nil
}
