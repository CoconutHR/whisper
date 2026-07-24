package web

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"whisper/internal/chat"
)

func (s *Server) handleConversationMessages(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	conversation := strings.TrimSpace(r.URL.Query().Get("conversation"))
	if conversation == "" {
		writeError(w, http.StatusBadRequest, errors.New("会话不能为空"))
		return
	}
	beforeSentAt := strings.TrimSpace(r.URL.Query().Get("beforeSentAt"))
	beforeID := strings.TrimSpace(r.URL.Query().Get("beforeId"))
	var cursor *chat.MessageCursor
	if beforeSentAt != "" || beforeID != "" {
		if beforeSentAt == "" || beforeID == "" {
			writeError(w, http.StatusBadRequest, errors.New("消息游标不完整"))
			return
		}
		parsed, err := time.Parse(time.RFC3339Nano, beforeSentAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, errors.New("消息游标无效"))
			return
		}
		cursor = &chat.MessageCursor{SentAt: parsed, MessageID: beforeID}
	}
	page, err := s.store.ConversationMessages(userID, conversation, cursor, chat.MessagePageSize)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"messages": page.Messages,
		"hasMore":  page.HasMore,
	})
}

func (s *Server) handleRecallCommand(userID, requestID, messageID string) {
	result, err := s.store.RecallMessage(userID, messageID)
	if err != nil {
		s.hub.sendTo(userID, map[string]any{
			"type": "error", "requestId": requestID, "message": err.Error(),
		})
		return
	}
	message := result.Message
	if message.Kind == "group" {
		for _, viewerID := range result.ViewerIDs {
			event := map[string]any{
				"type": "message_recalled", "conversation": chat.GroupConversationKey(message.GroupID),
				"messageId": message.ID,
			}
			if viewerID == userID && requestID != "" {
				event["requestId"] = requestID
			}
			s.hub.sendTo(viewerID, event)
		}
		return
	}
	targetName := s.store.NameForID(message.ToID)
	senderEvent := map[string]any{
		"type": "message_recalled", "conversation": "dm:" + targetName, "messageId": message.ID,
	}
	if requestID != "" {
		senderEvent["requestId"] = requestID
	}
	s.hub.sendTo(userID, senderEvent)
	if message.ToID == chat.CocoID {
		return
	}
	s.hub.sendTo(message.ToID, map[string]any{
		"type": "message_recalled", "conversation": "dm:" + s.store.NameForID(userID),
		"messageId": message.ID,
	})
}
