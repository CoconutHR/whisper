package web

import "whisper/internal/chat"

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
