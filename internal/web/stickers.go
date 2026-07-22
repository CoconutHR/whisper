package web

import (
	"errors"
	"net/http"
	"strings"

	"whisper/internal/chat"
)

func (s *Server) handleStickers(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method == http.MethodGet {
		stickers, err := s.store.Stickers(userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, stickers)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
		return
	}
	var request struct {
		AttachmentID string `json:"attachmentId"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	sticker, err := s.store.AddStickerDraft(userID, request.AttachmentID)
	if err != nil {
		writeStickerError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sticker)
}

func (s *Server) handleStickerFavorite(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var request struct {
		AttachmentID string `json:"attachmentId"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	sticker, err := s.store.FavoriteSticker(userID, request.AttachmentID)
	if err != nil {
		writeStickerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sticker)
}

func (s *Server) handleStickerItem(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w, http.MethodDelete)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/stickers/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	if err := s.store.RemoveSticker(userID, id); err != nil {
		writeStickerError(w, err)
		return
	}
	go s.cleanupExpiredAttachmentDrafts()
	w.WriteHeader(http.StatusNoContent)
}

func writeStickerError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, chat.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, chat.ErrAttachmentForbidden):
		writeError(w, http.StatusForbidden, err)
	default:
		writeError(w, http.StatusBadRequest, err)
	}
}
