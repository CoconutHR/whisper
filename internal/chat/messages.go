package chat

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrMessageForbidden = errors.New("没有权限操作这条消息")
	ErrRecallExpired    = errors.New("消息发送超过两分钟，不能撤回")
)

type RecallResult struct {
	Message   *Message
	ViewerIDs []string
}

func (s *Store) RecallMessage(userID, messageID string) (RecallResult, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return RecallResult{}, err
	}
	defer tx.Rollback()
	message, err := scanMessage(tx.QueryRow(`SELECT id, kind, from_id, COALESCE(to_id, ''),
		COALESCE(group_id, ''), text, sticker, sent_at, delivered_at, recalled_at
		FROM messages WHERE id = ?`, messageID))
	if errors.Is(err, sql.ErrNoRows) {
		return RecallResult{}, ErrNotFound
	}
	if err != nil {
		return RecallResult{}, err
	}
	if message.FromID != userID || message.FromID == "*" || message.RecalledAt != nil {
		return RecallResult{}, ErrMessageForbidden
	}
	age := time.Since(message.SentAt)
	if age < 0 || age > RecallWindow {
		return RecallResult{}, ErrRecallExpired
	}
	now := time.Now()
	result, err := tx.Exec(`UPDATE messages SET recalled_at = ? WHERE id = ? AND from_id = ? AND recalled_at IS NULL`,
		now.Format(time.RFC3339Nano), message.ID, userID)
	if err != nil {
		return RecallResult{}, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return RecallResult{}, ErrMessageForbidden
	}
	viewerIDs := []string{userID}
	if message.Kind == "group" {
		viewerIDs, err = groupMemberIDsTx(tx, message.GroupID)
		if err != nil {
			return RecallResult{}, err
		}
		foundSender := false
		for _, viewerID := range viewerIDs {
			foundSender = foundSender || viewerID == userID
		}
		if !foundSender {
			viewerIDs = append(viewerIDs, userID)
		}
	} else if message.ToID != "" && message.ToID != CocoID {
		viewerIDs = append(viewerIDs, message.ToID)
	}
	if err := tx.Commit(); err != nil {
		return RecallResult{}, err
	}
	message.RecalledAt = &now
	return RecallResult{Message: message, ViewerIDs: viewerIDs}, nil
}
