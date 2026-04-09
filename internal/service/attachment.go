package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"fresnel/internal/authz"
	"fresnel/internal/clamav"
	"fresnel/internal/domain"
	"fresnel/internal/storage"

	"github.com/google/uuid"
)

const maxAttachmentsPerEvent = 10

type AttachmentService struct {
	attachments storage.AttachmentStore
	events      storage.EventStore
	sectors     storage.SectorStore
	tlpRed      storage.TLPRedStore
	scanner     *clamav.Client
	authz       authz.Authorizer
	audit       *AuditService
	storageDir  string
}

func NewAttachmentService(
	attachments storage.AttachmentStore,
	events storage.EventStore,
	sectors storage.SectorStore,
	tlpRed storage.TLPRedStore,
	scanner *clamav.Client,
	az authz.Authorizer,
	audit *AuditService,
	storageDir string,
) *AttachmentService {
	return &AttachmentService{
		attachments: attachments, events: events, sectors: sectors,
		tlpRed: tlpRed, scanner: scanner, authz: az, audit: audit, storageDir: storageDir,
	}
}

func (s *AttachmentService) Upload(ctx context.Context, auth *domain.AuthContext, eventID uuid.UUID, filename, contentType string, size int64, body io.Reader) (*domain.Attachment, error) {
	event, err := s.events.GetByID(ctx, eventID)
	if err != nil || event == nil {
		return nil, ErrNotFound
	}

	sector, _ := s.sectors.GetByID(ctx, event.SectorContext)
	ancestry := ""
	if sector != nil {
		ancestry = sector.AncestryPath
	}
	recipients, _ := s.tlpRed.GetRecipients(ctx, "event", event.ID)
	res := authz.EventResource(event, ancestry, recipients)
	if !s.authz.Authorize(ctx, auth, authz.ActionEdit, res) {
		return nil, ErrForbidden
	}

	count, err := s.attachments.CountByEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if count >= maxAttachmentsPerEvent {
		return nil, fmt.Errorf("%w: maximum %d attachments per event", ErrValidation, maxAttachmentsPerEvent)
	}

	att := &domain.Attachment{
		ID:          uuid.New(),
		EventID:     eventID,
		Filename:    filepath.Base(filename),
		ContentType: contentType,
		SizeBytes:   size,
		ScanStatus:  domain.ScanPending,
		UploadedBy:  auth.UserID,
	}

	tmpPath := filepath.Join(s.storageDir, "tmp", att.ID.String())
	if err := os.MkdirAll(filepath.Dir(tmpPath), 0o750); err != nil {
		return nil, fmt.Errorf("create tmp dir: %w", err)
	}
	f, err := os.Create(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("create tmp file: %w", err)
	}
	if _, err := io.Copy(f, body); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return nil, fmt.Errorf("write tmp file: %w", err)
	}
	f.Close()

	if s.scanner != nil {
		scanFile, err := os.Open(tmpPath)
		if err != nil {
			os.Remove(tmpPath)
			return nil, err
		}
		result, err := s.scanner.ScanStream(ctx, scanFile)
		scanFile.Close()
		if err != nil {
			os.Remove(tmpPath)
			return nil, ErrScanUnavailable
		}
		if !result.Clean {
			os.Remove(tmpPath)
			att.ScanStatus = domain.ScanQuarantined
			s.audit.Log(ctx, auth, "quarantine", "attachment", &att.ID, domain.SeverityHigh, map[string]any{
				"filename": att.Filename, "virus": result.Virus, "event_id": eventID,
			})
			return nil, fmt.Errorf("%w: virus detected: %s", ErrQuarantined, result.Virus)
		}
		att.ScanStatus = domain.ScanClean
	} else {
		att.ScanStatus = domain.ScanClean
	}

	permPath := filepath.Join(s.storageDir, "files", att.ID.String())
	if err := os.MkdirAll(filepath.Dir(permPath), 0o750); err != nil {
		os.Remove(tmpPath)
		return nil, err
	}
	if err := os.Rename(tmpPath, permPath); err != nil {
		os.Remove(tmpPath)
		return nil, err
	}
	att.StoragePath = permPath

	if err := s.attachments.Create(ctx, att); err != nil {
		os.Remove(permPath)
		return nil, err
	}

	s.audit.Log(ctx, auth, "upload", "attachment", &att.ID, domain.SeverityInfo, map[string]any{
		"filename": att.Filename, "event_id": eventID, "size": att.SizeBytes,
	})
	return att, nil
}

func (s *AttachmentService) Download(ctx context.Context, auth *domain.AuthContext, id uuid.UUID) (*domain.Attachment, io.ReadCloser, error) {
	att, err := s.attachments.GetByID(ctx, id)
	if err != nil || att == nil {
		return nil, nil, ErrNotFound
	}
	event, err := s.events.GetByID(ctx, att.EventID)
	if err != nil || event == nil {
		return nil, nil, ErrNotFound
	}

	sector, _ := s.sectors.GetByID(ctx, event.SectorContext)
	ancestry := ""
	if sector != nil {
		ancestry = sector.AncestryPath
	}
	recipients, _ := s.tlpRed.GetRecipients(ctx, "event", event.ID)
	res := authz.EventResource(event, ancestry, recipients)
	if !s.authz.Authorize(ctx, auth, authz.ActionView, res) {
		return nil, nil, ErrForbidden
	}

	f, err := os.Open(att.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("open file: %w", err)
	}
	return att, f, nil
}

func (s *AttachmentService) ListByEvent(ctx context.Context, auth *domain.AuthContext, eventID uuid.UUID) ([]*domain.Attachment, error) {
	event, err := s.events.GetByID(ctx, eventID)
	if err != nil || event == nil {
		return nil, ErrNotFound
	}
	sector, _ := s.sectors.GetByID(ctx, event.SectorContext)
	ancestry := ""
	if sector != nil {
		ancestry = sector.AncestryPath
	}
	recipients, _ := s.tlpRed.GetRecipients(ctx, "event", event.ID)
	res := authz.EventResource(event, ancestry, recipients)
	if !s.authz.Authorize(ctx, auth, authz.ActionView, res) {
		return nil, ErrForbidden
	}
	return s.attachments.ListByEvent(ctx, eventID)
}

func (s *AttachmentService) Delete(ctx context.Context, auth *domain.AuthContext, id uuid.UUID) error {
	att, err := s.attachments.GetByID(ctx, id)
	if err != nil || att == nil {
		return ErrNotFound
	}
	event, err := s.events.GetByID(ctx, att.EventID)
	if err != nil || event == nil {
		return ErrNotFound
	}
	sector, _ := s.sectors.GetByID(ctx, event.SectorContext)
	ancestry := ""
	if sector != nil {
		ancestry = sector.AncestryPath
	}
	res := authz.EventResource(event, ancestry, nil)
	if !s.authz.Authorize(ctx, auth, authz.ActionDelete, res) {
		return ErrForbidden
	}
	if err := s.attachments.Delete(ctx, id); err != nil {
		return err
	}
	os.Remove(att.StoragePath)
	s.audit.Log(ctx, auth, "delete", "attachment", &id, domain.SeverityInfo, map[string]any{
		"filename": att.Filename, "event_id": att.EventID,
	})
	return nil
}
