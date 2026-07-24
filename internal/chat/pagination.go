package chat

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

const MessagePageSize = 50

type MessageCursor struct {
	SentAt    time.Time
	MessageID string
}

type MessagePage struct {
	Messages []MessageView
	HasMore  bool
}

func conversationMessageFilter(q queryer, userID string, conversation resolvedConversation) (string, []any, error) {
	visibleAfter, err := clearedTime(q, userID, conversation.StableKey)
	if err != nil {
		return "", nil, err
	}
	if conversation.GroupID != "" && conversation.HistoryFrom.After(visibleAfter) {
		visibleAfter = conversation.HistoryFrom
	}
	if conversation.GroupID != "" {
		return "kind = 'group' AND group_id = ? AND sent_at > ?",
			[]any{conversation.GroupID, visibleAfter.Format(time.RFC3339Nano)}, nil
	}
	return `kind = 'private' AND ((from_id = ? AND to_id = ?) OR (from_id = ? AND to_id = ?))
		AND sent_at > ?`, []any{
			userID, conversation.TargetID, conversation.TargetID, userID,
			visibleAfter.Format(time.RFC3339Nano),
		}, nil
}

func conversationMessagePageTx(tx *sql.Tx, userID string, conversation resolvedConversation,
	users map[string]*databaseUser, before *MessageCursor, limit int) (MessagePage, error) {
	if limit <= 0 || limit > MessagePageSize {
		limit = MessagePageSize
	}
	filter, args, err := conversationMessageFilter(tx, userID, conversation)
	if err != nil {
		return MessagePage{}, err
	}
	if before != nil {
		if before.SentAt.IsZero() || strings.TrimSpace(before.MessageID) == "" {
			return MessagePage{}, errors.New("消息游标无效")
		}
		filter += " AND (sent_at < ? OR (sent_at = ? AND id < ?))"
		value := before.SentAt.Format(time.RFC3339Nano)
		args = append(args, value, value, before.MessageID)
	}
	args = append(args, limit+1)
	rows, err := tx.Query(`SELECT id, kind, from_id, COALESCE(to_id, ''), COALESCE(group_id, ''),
		text, sticker, sent_at, delivered_at, recalled_at FROM messages WHERE `+filter+`
		ORDER BY sent_at DESC, id DESC LIMIT ?`, args...)
	if err != nil {
		return MessagePage{}, err
	}
	messages := make([]*Message, 0, limit+1)
	for rows.Next() {
		message, scanErr := scanMessage(rows)
		if scanErr != nil {
			rows.Close()
			return MessagePage{}, scanErr
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return MessagePage{}, err
	}
	if err := rows.Close(); err != nil {
		return MessagePage{}, err
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	if err := hydrateMessageRelations(tx, messages); err != nil {
		return MessagePage{}, err
	}
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	page := MessagePage{Messages: make([]MessageView, 0, len(messages)), HasMore: hasMore}
	for _, message := range messages {
		page.Messages = append(page.Messages, messageView(userID, message, users))
	}
	return page, nil
}

func hydrateMessageRelations(q queryer, messages []*Message) error {
	if len(messages) == 0 {
		return nil
	}
	byID := make(map[string]*Message, len(messages))
	placeholders := make([]string, 0, len(messages))
	args := make([]any, 0, len(messages)*2)
	for _, message := range messages {
		byID[message.ID] = message
		placeholders = append(placeholders, "?")
		args = append(args, message.ID)
	}
	relationArgs := append([]any(nil), args...)
	args = append(args, relationArgs...)
	list := strings.Join(placeholders, ",")
	rows, err := q.Query(`SELECT 'attachment', ma.message_id, ma.position, a.id, a.uploader_id,
		a.object_key, a.original_name, a.content_type, a.size, a.status, a.created_at
		FROM message_attachments ma JOIN attachments a ON a.id = ma.attachment_id
		WHERE ma.message_id IN (`+list+`)
		UNION ALL
		SELECT 'sticker', ms.message_id, 0, a.id, a.uploader_id, a.object_key, a.original_name,
		a.content_type, a.size, a.status, a.created_at
		FROM message_stickers ms JOIN attachments a ON a.id = ms.attachment_id
		WHERE ms.message_id IN (`+list+`) ORDER BY 2, 3`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var relation, messageID, createdAt string
		var position int
		var attachment Attachment
		if err := rows.Scan(&relation, &messageID, &position, &attachment.ID, &attachment.UploaderID,
			&attachment.ObjectKey, &attachment.Name, &attachment.ContentType, &attachment.Size,
			&attachment.Status, &createdAt); err != nil {
			return err
		}
		attachment.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return err
		}
		message := byID[messageID]
		if message == nil {
			continue
		}
		view := attachmentView(attachment)
		if relation == "sticker" {
			message.StickerAttachment = &view
		} else {
			message.Attachments = append(message.Attachments, view)
		}
	}
	return rows.Err()
}

func conversationUnreadCountTx(tx *sql.Tx, userID string, conversation resolvedConversation,
	readCursor conversationReadCursor, hasReadCursor bool) (int, error) {
	filter, args, err := conversationMessageFilter(tx, userID, conversation)
	if err != nil {
		return 0, err
	}
	filter += " AND from_id <> ? AND from_id <> '*'"
	args = append(args, userID)
	if hasReadCursor {
		value := readCursor.SentAt.Format(time.RFC3339Nano)
		filter += " AND (sent_at > ? OR (sent_at = ? AND id > ?))"
		args = append(args, value, value, readCursor.MessageID)
	}
	var count int
	if err := tx.QueryRow("SELECT COUNT(*) FROM messages WHERE "+filter, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) ConversationMessages(userID, clientKey string, before *MessageCursor, limit int) (MessagePage, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return MessagePage{}, err
	}
	defer tx.Rollback()
	viewer, err := userByID(tx, userID)
	if err != nil {
		return MessagePage{}, err
	}
	if viewer == nil {
		return MessagePage{}, ErrNotFound
	}
	conversation, err := resolveConversationTx(tx, userID, clientKey)
	if err != nil {
		return MessagePage{}, err
	}
	users, err := allUsers(tx)
	if err != nil {
		return MessagePage{}, err
	}
	usersByID := make(map[string]*databaseUser, len(users))
	for _, user := range users {
		usersByID[user.ID] = user
	}
	page, err := conversationMessagePageTx(tx, userID, conversation, usersByID, before, limit)
	if err != nil {
		return MessagePage{}, err
	}
	if err := tx.Commit(); err != nil {
		return MessagePage{}, err
	}
	return page, nil
}
