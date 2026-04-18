package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"fresnel/internal/service"
)

type AttachmentHandler struct {
	attachments *service.AttachmentService
}

func NewAttachmentHandler(attachments *service.AttachmentService) *AttachmentHandler {
	return &AttachmentHandler{attachments: attachments}
}

func (h *AttachmentHandler) Upload(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	eventID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}

	const maxUpload = 50 << 20 // 50 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, r, fmt.Errorf("%w: %v", service.ErrValidation, err))
		return
	}
	defer file.Close()

	att, err := h.attachments.Upload(
		r.Context(), auth, eventID,
		header.Filename, header.Header.Get("Content-Type"), header.Size, file,
	)
	if err != nil {
		respondError(w, r, err)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/events/"+eventID.String())
		w.WriteHeader(http.StatusCreated)
		return
	}
	respondJSON(w, http.StatusCreated, att)
}

func (h *AttachmentHandler) Download(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "attachmentId")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	att, reader, err := h.attachments.Download(r.Context(), auth, id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", att.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, att.Filename))
	if att.SizeBytes > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(att.SizeBytes, 10))
	}
	w.WriteHeader(http.StatusOK)
	// io.Copy is done via http.ServeContent-style streaming
	buf := make([]byte, 32*1024)
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return
			}
		}
		if readErr != nil {
			return
		}
	}
}

func (h *AttachmentHandler) ListByEvent(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	eventID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	attachments, err := h.attachments.ListByEvent(r.Context(), auth, eventID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, attachments)
}

func (h *AttachmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	auth := getAuth(r)
	id, err := parseUUID(r, "attachmentId")
	if err != nil {
		respondError(w, r, service.ErrValidation)
		return
	}
	if err := h.attachments.Delete(r.Context(), auth, id); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
