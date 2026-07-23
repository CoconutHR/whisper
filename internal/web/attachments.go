package web

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"strings"
	"time"

	"whisper/internal/chat"
)

const (
	attachmentURLTTL = 5 * time.Minute
	previewURLTTL    = 2 * time.Hour
)

func (s *Server) requireObjectStore(w http.ResponseWriter) bool {
	if s.objects != nil {
		return true
	}
	writeError(w, http.StatusServiceUnavailable, errors.New("文件上传尚未配置"))
	return false
}

func (s *Server) cleanupExpiredAttachmentDrafts() {
	if s.objects == nil {
		return
	}
	attachments, err := s.store.ExpiredAttachmentDrafts(time.Now().Add(-24 * time.Hour))
	if err != nil {
		return
	}
	for _, attachment := range attachments {
		if err := s.objects.Delete(context.Background(), attachment.ObjectKey); err != nil {
			continue
		}
		_ = s.store.DeleteExpiredAttachmentDraft(attachment.ID)
	}
}

func (s *Server) handleAttachmentPresign(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if !s.requireObjectStore(w) {
		return
	}
	var request struct {
		Name        string `json:"name"`
		ContentType string `json:"contentType"`
		Size        int64  `json:"size"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	attachment, err := s.store.CreateAttachmentDraft(userID, request.Name, request.ContentType, request.Size)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	presigned, err := s.objects.PresignPut(r.Context(), attachment.ObjectKey, attachment.ContentType, attachment.Size, attachmentURLTTL)
	if err != nil {
		_, _ = s.store.DeleteAttachmentDraft(userID, attachment.ID)
		writeError(w, http.StatusBadGateway, err)
		return
	}
	go s.cleanupExpiredAttachmentDrafts()
	writeJSON(w, http.StatusCreated, map[string]any{
		"attachmentId": attachment.ID,
		"uploadUrl":    presigned.URL,
		"headers":      presigned.Headers,
		"expiresAt":    presigned.ExpiresAt,
	})
}

func (s *Server) handleAttachmentComplete(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if !s.requireObjectStore(w) {
		return
	}
	var request struct {
		AttachmentID string `json:"attachmentId"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	attachment, err := s.store.OwnedAttachmentDraft(userID, request.AttachmentID)
	if err != nil {
		writeAttachmentError(w, err)
		return
	}
	metadata, err := s.objects.Head(r.Context(), attachment.ObjectKey)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	draft := attachment
	attachment, err = s.store.CompleteAttachmentDraft(userID, draft.ID, metadata.Size, metadata.ContentType)
	if err != nil {
		if deleteErr := s.objects.Delete(r.Context(), draft.ObjectKey); deleteErr != nil {
			writeError(w, http.StatusBadGateway, deleteErr)
			return
		}
		_, _ = s.store.DeleteAttachmentDraft(userID, draft.ID)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"attachmentId": attachment.ID, "status": "ready"})
}

func (s *Server) handleAttachmentItem(w http.ResponseWriter, r *http.Request, userID string) {
	if !s.requireObjectStore(w) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/attachments/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodDelete {
		attachment, err := s.store.OwnedAttachmentDraft(userID, id)
		if err != nil {
			writeAttachmentError(w, err)
			return
		}
		if err := s.objects.Delete(r.Context(), attachment.ObjectKey); err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		if _, err := s.store.DeleteAttachmentDraft(userID, id); err != nil {
			writeAttachmentError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, http.MethodGet, http.MethodHead, http.MethodDelete)
		return
	}
	attachment, err := s.store.AttachmentForViewer(userID, id)
	if err != nil {
		writeAttachmentError(w, err)
		return
	}
	disposition, ttl := attachmentResponsePolicy(attachment.ContentType, r.URL.Query().Get("download") == "1")
	contentDisposition := mime.FormatMediaType(disposition, map[string]string{"filename": attachment.Name})
	presigned, err := s.objects.PresignGet(r.Context(), attachment.ObjectKey, contentDisposition, ttl)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	http.Redirect(w, r, presigned.URL, http.StatusTemporaryRedirect)
}

func attachmentResponsePolicy(contentType string, download bool) (string, time.Duration) {
	disposition := "attachment"
	if !download && chat.IsBrowserPreviewableContentType(contentType) {
		disposition = "inline"
	}
	ttl := attachmentURLTTL
	if chat.IsStreamableMediaContentType(contentType) || chat.IsBrowserDocumentContentType(contentType) {
		ttl = previewURLTTL
	}
	return disposition, ttl
}

func writeAttachmentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, chat.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, chat.ErrAttachmentForbidden):
		writeError(w, http.StatusForbidden, err)
	default:
		writeError(w, http.StatusBadRequest, err)
	}
}
