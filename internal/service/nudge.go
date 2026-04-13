package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"fresnel/internal/domain"
	"fresnel/internal/mail"
	"fresnel/internal/storage"

	"github.com/google/uuid"
)

const (
	nudgeTypeDaily      = "DAILY"
	nudgeTypeEscalation = "ESCALATION"
	nudgeTypeDigest     = "DIGEST"
)

// EscalationResetter clears escalation state when a new event update is recorded.
type EscalationResetter interface {
	ResetEscalation(ctx context.Context, eventID uuid.UUID) error
}

type noopEscalationReset struct{}

func (noopEscalationReset) ResetEscalation(context.Context, uuid.UUID) error { return nil }

// NudgeService schedules impact-based reminders and hierarchical escalation emails.
type NudgeService struct {
	nudges  storage.NudgeStore
	events  storage.EventStore
	updates storage.EventUpdateStore
	users   storage.UserStore
	roles   storage.RoleStore
	orgs    storage.OrganizationStore
	sectors storage.SectorStore
	mailer  mail.Sender
	audit   *AuditService
	logger  *slog.Logger
	appURL  string

	stopCh         chan struct{}
	done           chan struct{}
	ticker         *time.Ticker
	mu             sync.Mutex
	lastDigestWeek int
}

// NewNudgeService builds a nudge scheduler. Extra stores (updates, roles, orgs, sectors)
// are required for escalation and last-activity detection.
func NewNudgeService(
	nudges storage.NudgeStore,
	events storage.EventStore,
	updates storage.EventUpdateStore,
	users storage.UserStore,
	roles storage.RoleStore,
	orgs storage.OrganizationStore,
	sectors storage.SectorStore,
	mailer mail.Sender,
	audit *AuditService,
	logger *slog.Logger,
	appURL string,
) *NudgeService {
	return &NudgeService{
		nudges:  nudges,
		events:  events,
		updates: updates,
		users:   users,
		roles:   roles,
		orgs:    orgs,
		sectors: sectors,
		mailer:  mailer,
		audit:   audit,
		logger:  logger,
		appURL:  strings.TrimRight(appURL, "/"),
	}
}

// Start begins the 15-minute ticker loop. ctx is used for the initial tick only; subsequent
// ticks use context.Background().
func (s *NudgeService) Start(ctx context.Context) {
	s.mu.Lock()
	if s.stopCh != nil {
		s.mu.Unlock()
		return
	}
	s.stopCh = make(chan struct{})
	s.done = make(chan struct{})
	s.ticker = time.NewTicker(15 * time.Minute)
	s.mu.Unlock()

	go func() {
		defer close(s.done)
		defer s.ticker.Stop()

		s.tick(ctx)

		for {
			select {
			case <-s.stopCh:
				return
			case <-s.ticker.C:
				s.tick(context.Background())
			}
		}
	}()
}

// Stop shuts down the scheduler; it blocks until the goroutine exits.
func (s *NudgeService) Stop() {
	s.mu.Lock()
	ch := s.stopCh
	s.mu.Unlock()
	if ch == nil {
		return
	}
	close(ch)
	<-s.done
	s.mu.Lock()
	s.stopCh = nil
	s.done = nil
	s.ticker = nil
	s.mu.Unlock()
}

func (s *NudgeService) tick(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	now := time.Now().UTC()
	if now.Weekday() == time.Monday {
		y, w := now.ISOWeek()
		key := y*100 + w
		s.mu.Lock()
		if s.lastDigestWeek != key {
			s.lastDigestWeek = key
			s.mu.Unlock()
			if err := s.SendWeeklyDigest(ctx); err != nil {
				s.logger.Error("weekly digest failed", "err", err)
			}
		} else {
			s.mu.Unlock()
		}
	}

	open, err := s.listOpenEvents(ctx)
	if err != nil {
		s.logger.Error("nudge tick list open events", "err", err)
		return
	}

	for _, ev := range open {
		if err := s.checkEscalation(ctx, ev); err != nil {
			s.logger.Error("check escalation", "event_id", ev.ID, "err", err)
		}
	}

	bySubmitter := make(map[uuid.UUID][]*domain.Event)
	for _, ev := range open {
		if !impactWantsScheduledNudge(ev.Impact) {
			continue
		}
		bySubmitter[ev.SubmitterID] = append(bySubmitter[ev.SubmitterID], ev)
	}

	for uid, evs := range bySubmitter {
		u, err := s.users.GetByID(ctx, uid)
		if err != nil || u == nil {
			s.logger.Warn("nudge skip user", "user_id", uid, "err", err)
			continue
		}
		if u.Email == "" {
			continue
		}
		s.processUser(ctx, u, evs)
	}
}

func impactWantsScheduledNudge(i domain.Impact) bool {
	switch i {
	case domain.ImpactCritical, domain.ImpactHigh, domain.ImpactModerate:
		return true
	default:
		return false
	}
}

func (s *NudgeService) listOpenEvents(ctx context.Context) ([]*domain.Event, error) {
	const pageSize = 500
	var all []*domain.Event
	offset := 0
	for {
		res, err := s.events.List(ctx, domain.EventFilter{
			Pagination: domain.Pagination{Offset: offset, Limit: pageSize},
		})
		if err != nil {
			return nil, err
		}
		for _, e := range res.Items {
			if e.Status.IsOpen() {
				all = append(all, e)
			}
		}
		if len(res.Items) < pageSize {
			break
		}
		offset += pageSize
	}
	return all, nil
}

// processUser applies impact-based cadence rules for the submitter's open events.
func (s *NudgeService) processUser(ctx context.Context, user *domain.User, events []*domain.Event) {
	for _, ev := range events {
		switch ev.Impact {
		case domain.ImpactCritical:
			ok, err := s.nudges.HasNudgeToday(ctx, ev.ID, user.ID)
			if err != nil {
				s.logger.Error("HasNudgeToday", "err", err)
				continue
			}
			if ok {
				continue
			}
			if err := s.sendNudge(ctx, user, ev, nudgeTypeDaily, 0); err != nil {
				s.logger.Error("send critical nudge", "event_id", ev.ID, "err", err)
			}
		case domain.ImpactHigh:
			if !s.shouldIntervalNudge(ctx, ev.ID, user.ID, 48*time.Hour) {
				continue
			}
			if err := s.sendNudge(ctx, user, ev, nudgeTypeDaily, 0); err != nil {
				s.logger.Error("send high nudge", "event_id", ev.ID, "err", err)
			}
		case domain.ImpactModerate:
			if !s.shouldIntervalNudge(ctx, ev.ID, user.ID, 7*24*time.Hour) {
				continue
			}
			if err := s.sendNudge(ctx, user, ev, nudgeTypeDaily, 0); err != nil {
				s.logger.Error("send moderate nudge", "event_id", ev.ID, "err", err)
			}
		}
	}
}

func (s *NudgeService) shouldIntervalNudge(ctx context.Context, eventID, recipientID uuid.UUID, minGap time.Duration) bool {
	t, ok, err := s.nudges.LastNudgeSentAt(ctx, eventID, recipientID)
	if err != nil {
		s.logger.Error("LastNudgeSentAt", "err", err)
		return false
	}
	if !ok {
		return true
	}
	return time.Since(t) >= minGap
}

func (s *NudgeService) sendNudge(ctx context.Context, user *domain.User, event *domain.Event, nudgeType string, level int) error {
	var subject, body string
	switch nudgeType {
	case nudgeTypeDaily:
		subject = fmt.Sprintf("Action required: %s (Impact: %s)", event.Title, event.Impact)
		body = fmt.Sprintf(`You have an open event that needs attention.

Title: %s
Impact: %s
Link: %s

This is an automated message from Fresnel.`,
			event.Title, event.Impact, s.eventLink(event.ID))
	case nudgeTypeEscalation:
		subject = fmt.Sprintf("Escalation: %s requires attention (Level %d)", event.Title, level)
		body = fmt.Sprintf(`An open event has not received a timely update and has been escalated.

Title: %s
Escalation level: %d
Link: %s

This is an automated message from Fresnel.`,
			event.Title, level, s.eventLink(event.ID))
	default:
		subject = fmt.Sprintf("Fresnel notification: %s", event.Title)
		body = s.eventLink(event.ID)
	}

	if err := s.mailer.Send(ctx, user.Email, subject, body); err != nil {
		return err
	}
	if err := s.nudges.LogNudge(ctx, event.ID, user.ID, nudgeType, level); err != nil {
		return fmt.Errorf("log nudge: %w", err)
	}

	sys := &domain.AuthContext{
		UserID:      uuid.Nil,
		DisplayName: "nudge-scheduler",
		Email:       "nudge-scheduler@fresnel.local",
	}
	s.audit.Log(ctx, sys, "nudge_sent", "event", &event.ID, domain.SeverityInfo, map[string]any{
		"nudge_type":    nudgeType,
		"recipient_id":  user.ID.String(),
		"escalation_lv": level,
	})
	return nil
}

func (s *NudgeService) eventLink(eventID uuid.UUID) string {
	return fmt.Sprintf("%s/api/v1/events/%s", s.appURL, eventID.String())
}

// checkEscalation advances the escalation chain when there has been no event update for over
// 24 hours (PoC: simple wall-clock interval, not business days). At most one escalation step
// runs per 24 hours per event.
func (s *NudgeService) checkEscalation(ctx context.Context, event *domain.Event) error {
	if !event.Status.IsOpen() {
		return nil
	}

	lastAct := s.lastEventActivity(ctx, event)
	if time.Since(lastAct) < 24*time.Hour {
		return nil
	}

	lastEsc, hadEsc, err := s.nudges.LastEscalationNudgeTime(ctx, event.ID)
	if err != nil {
		return err
	}
	if hadEsc && time.Since(lastEsc) < 24*time.Hour {
		return nil
	}

	level, _, err := s.nudges.GetEscalationState(ctx, event.ID)
	if err != nil {
		return err
	}

	nextLevel := level + 1
	recipients, err := s.escalationRecipients(ctx, event, nextLevel)
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		return nil
	}

	if err := s.nudges.SetEscalationLevel(ctx, event.ID, nextLevel); err != nil {
		return err
	}

	for _, u := range recipients {
		if u == nil || u.Email == "" {
			continue
		}
		if err := s.sendNudge(ctx, u, event, nudgeTypeEscalation, nextLevel); err != nil {
			s.logger.Error("escalation nudge", "event_id", event.ID, "user_id", u.ID, "err", err)
		}
	}
	return nil
}

func (s *NudgeService) lastEventActivity(ctx context.Context, event *domain.Event) time.Time {
	t := event.CreatedAt
	if event.UpdatedAt.After(t) {
		t = event.UpdatedAt
	}
	if u, ok, err := s.updates.LatestCreatedAt(ctx, event.ID); err == nil && ok && u.After(t) {
		t = u
	}
	return t
}

// Level 1 = org root; level 2+ = sector roots walking up sector ancestry; beyond the tree = platform root.
func (s *NudgeService) escalationRecipients(ctx context.Context, event *domain.Event, level int) ([]*domain.User, error) {
	if level < 1 {
		return nil, nil
	}
	if level == 1 {
		return s.userForOrgRoot(ctx, event.OrganizationID)
	}

	org, err := s.orgs.GetByID(ctx, event.OrganizationID)
	if err != nil || org == nil {
		return nil, err
	}
	secID := org.SectorID
	for step := 2; step < level; step++ {
		sec, err := s.sectors.GetByID(ctx, secID)
		if err != nil || sec == nil {
			return s.userForPlatformRoot(ctx)
		}
		if sec.ParentSectorID == nil {
			return s.userForPlatformRoot(ctx)
		}
		secID = *sec.ParentSectorID
	}
	rid, err := s.roles.GetRoot(ctx, domain.ScopeSector, &secID)
	if err != nil {
		return nil, err
	}
	if rid == nil {
		return s.userForPlatformRoot(ctx)
	}
	u, err := s.users.GetByID(ctx, *rid)
	if err != nil || u == nil {
		return s.userForPlatformRoot(ctx)
	}
	return []*domain.User{u}, nil
}

func (s *NudgeService) userForOrgRoot(ctx context.Context, orgID uuid.UUID) ([]*domain.User, error) {
	rid, err := s.roles.GetRoot(ctx, domain.ScopeOrg, &orgID)
	if err != nil {
		return nil, err
	}
	if rid == nil {
		return nil, nil
	}
	u, err := s.users.GetByID(ctx, *rid)
	if err != nil || u == nil {
		return nil, err
	}
	return []*domain.User{u}, nil
}

func (s *NudgeService) userForPlatformRoot(ctx context.Context) ([]*domain.User, error) {
	rid, err := s.roles.GetRoot(ctx, domain.ScopePlatform, nil)
	if err != nil {
		return nil, err
	}
	if rid == nil {
		return nil, nil
	}
	u, err := s.users.GetByID(ctx, *rid)
	if err != nil || u == nil {
		return nil, err
	}
	return []*domain.User{u}, nil
}

// ResetEscalation clears stored escalation state after a new event update (implements EscalationResetter).
func (s *NudgeService) ResetEscalation(ctx context.Context, eventID uuid.UUID) error {
	return s.nudges.ResetEscalation(ctx, eventID)
}

// SendWeeklyDigest emails each submitter who has open events (impact above INFO) with a count and list.
func (s *NudgeService) SendWeeklyDigest(ctx context.Context) error {
	open, err := s.listOpenEvents(ctx)
	if err != nil {
		return err
	}

	bySubmitter := make(map[uuid.UUID][]*domain.Event)
	for _, ev := range open {
		if ev.Impact == domain.ImpactInfo || ev.Impact == domain.ImpactLow {
			continue
		}
		bySubmitter[ev.SubmitterID] = append(bySubmitter[ev.SubmitterID], ev)
	}

	for uid, evs := range bySubmitter {
		u, err := s.users.GetByID(ctx, uid)
		if err != nil || u == nil || u.Email == "" {
			continue
		}
		n := len(evs)
		subject := fmt.Sprintf("Weekly Digest: %d open events require attention", n)
		var b strings.Builder
		b.WriteString(fmt.Sprintf("You have %d open event(s) that may need attention:\n\n", n))
		for _, ev := range evs {
			b.WriteString(fmt.Sprintf("- %s (%s) — %s\n", ev.Title, ev.Impact, s.eventLink(ev.ID)))
		}
		b.WriteString("\nThis is an automated weekly summary from Fresnel.\n")

		if err := s.mailer.Send(ctx, u.Email, subject, b.String()); err != nil {
			s.logger.Error("digest send", "user_id", uid, "err", err)
			continue
		}
		for _, ev := range evs {
			_ = s.nudges.LogNudge(ctx, ev.ID, u.ID, nudgeTypeDigest, 0)
		}
		sys := &domain.AuthContext{
			UserID:      uuid.Nil,
			DisplayName: "nudge-scheduler",
			Email:       "nudge-scheduler@fresnel.local",
		}
		s.audit.Log(ctx, sys, "weekly_digest", "user", &u.ID, domain.SeverityInfo, map[string]any{
			"event_count": n,
		})
	}
	return nil
}
