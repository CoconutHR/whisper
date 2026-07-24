package chat

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type conversationReadCursor struct {
	SentAt    time.Time
	MessageID string
}

type resolvedConversation struct {
	ClientKey   string
	StableKey   string
	GroupID     string
	TargetID    string
	HistoryFrom time.Time
}

func currentSchemaVersion(q queryer) (int, error) {
	var value string
	err := q.QueryRow(`SELECT value FROM meta WHERE key = 'schema_version'`).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	version, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("无效的数据库版本 %q: %w", value, err)
	}
	return version, nil
}

func cursorComesAfter(candidate, current conversationReadCursor) bool {
	return candidate.SentAt.After(current.SentAt) ||
		(candidate.SentAt.Equal(current.SentAt) && candidate.MessageID > current.MessageID)
}

func (s *Store) migrateConversationReads(previousVersion int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if previousVersion < 9 {
		if err := seedExistingConversationReadsTx(tx); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`INSERT INTO meta(key, value) VALUES ('schema_version', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, schemaVersion); err != nil {
		return err
	}
	return tx.Commit()
}

// Existing installations start from their current history so an upgrade does not
// turn every old message into an unread message.
func seedExistingConversationReadsTx(tx *sql.Tx) error {
	userRows, err := tx.Query(`SELECT id FROM users`)
	if err != nil {
		return err
	}
	users := map[string]bool{}
	for userRows.Next() {
		var userID string
		if err := userRows.Scan(&userID); err != nil {
			userRows.Close()
			return err
		}
		users[userID] = true
	}
	if err := userRows.Close(); err != nil {
		return err
	}

	memberRows, err := tx.Query(`SELECT group_id, user_id FROM group_members`)
	if err != nil {
		return err
	}
	groupMembers := map[string][]string{}
	for memberRows.Next() {
		var groupID, userID string
		if err := memberRows.Scan(&groupID, &userID); err != nil {
			memberRows.Close()
			return err
		}
		groupMembers[groupID] = append(groupMembers[groupID], userID)
	}
	if err := memberRows.Close(); err != nil {
		return err
	}

	type cursorKey struct {
		UserID          string
		ConversationKey string
	}
	latest := map[cursorKey]conversationReadCursor{}
	messageRows, err := tx.Query(`SELECT id, kind, from_id, COALESCE(to_id, ''), COALESCE(group_id, ''), sent_at FROM messages`)
	if err != nil {
		return err
	}
	for messageRows.Next() {
		var id, kind, fromID, toID, groupID, sentAtValue string
		if err := messageRows.Scan(&id, &kind, &fromID, &toID, &groupID, &sentAtValue); err != nil {
			messageRows.Close()
			return err
		}
		sentAt, err := time.Parse(time.RFC3339Nano, sentAtValue)
		if err != nil {
			messageRows.Close()
			return err
		}
		candidate := conversationReadCursor{SentAt: sentAt, MessageID: id}
		setLatest := func(userID, conversationKey string) {
			key := cursorKey{UserID: userID, ConversationKey: conversationKey}
			if current, ok := latest[key]; !ok || cursorComesAfter(candidate, current) {
				latest[key] = candidate
			}
		}
		if kind == "group" {
			if groupID == "" {
				groupID = PublicGroupID
			}
			for _, memberID := range groupMembers[groupID] {
				setLatest(memberID, GroupConversationKey(groupID))
			}
			continue
		}
		if users[fromID] {
			setLatest(fromID, "dm:"+toID)
		}
		if users[toID] {
			setLatest(toID, "dm:"+fromID)
		}
	}
	if err := messageRows.Close(); err != nil {
		return err
	}
	for key, cursor := range latest {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO conversation_reads(
			user_id, conversation_key, last_read_at, last_read_message_id
		) VALUES (?, ?, ?, ?)`, key.UserID, key.ConversationKey,
			cursor.SentAt.Format(time.RFC3339Nano), cursor.MessageID); err != nil {
			return err
		}
	}
	return nil
}

func conversationReadCursors(q queryer, userID string) (map[string]conversationReadCursor, error) {
	rows, err := q.Query(`SELECT conversation_key, last_read_at, last_read_message_id
		FROM conversation_reads WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]conversationReadCursor{}
	for rows.Next() {
		var key, sentAtValue, messageID string
		if err := rows.Scan(&key, &sentAtValue, &messageID); err != nil {
			return nil, err
		}
		sentAt, err := time.Parse(time.RFC3339Nano, sentAtValue)
		if err != nil {
			return nil, err
		}
		result[key] = conversationReadCursor{SentAt: sentAt, MessageID: messageID}
	}
	return result, rows.Err()
}

func stableConversationKeyForMessage(userID string, message *Message) string {
	if message.Kind == "group" {
		groupID := message.GroupID
		if groupID == "" {
			groupID = PublicGroupID
		}
		return GroupConversationKey(groupID)
	}
	counterpartID := message.FromID
	if counterpartID == userID {
		counterpartID = message.ToID
	}
	return "dm:" + counterpartID
}

func setConversationReadTx(tx *sql.Tx, userID, conversationKey string, cursor conversationReadCursor) error {
	var sentAtValue, messageID string
	err := tx.QueryRow(`SELECT last_read_at, last_read_message_id FROM conversation_reads
		WHERE user_id = ? AND conversation_key = ?`, userID, conversationKey).Scan(&sentAtValue, &messageID)
	if err == nil {
		sentAt, parseErr := time.Parse(time.RFC3339Nano, sentAtValue)
		if parseErr != nil {
			return parseErr
		}
		if !cursorComesAfter(cursor, conversationReadCursor{SentAt: sentAt, MessageID: messageID}) {
			return nil
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = tx.Exec(`INSERT INTO conversation_reads(user_id, conversation_key, last_read_at, last_read_message_id)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, conversation_key) DO UPDATE SET
		last_read_at = excluded.last_read_at, last_read_message_id = excluded.last_read_message_id`,
		userID, conversationKey, cursor.SentAt.Format(time.RFC3339Nano), cursor.MessageID)
	return err
}

func seedLatestConversationReadTx(tx *sql.Tx, userID, conversationKey string) error {
	var rows *sql.Rows
	var err error
	if strings.HasPrefix(conversationKey, "group:") {
		groupID := strings.TrimPrefix(conversationKey, "group:")
		rows, err = tx.Query(`SELECT id, sent_at FROM messages WHERE kind = 'group'
			AND COALESCE(NULLIF(group_id, ''), ?) = ?`, PublicGroupID, groupID)
	} else if strings.HasPrefix(conversationKey, "dm:") {
		targetID := strings.TrimPrefix(conversationKey, "dm:")
		rows, err = tx.Query(`SELECT id, sent_at FROM messages WHERE kind = 'private'
			AND ((from_id = ? AND to_id = ?) OR (from_id = ? AND to_id = ?))`,
			userID, targetID, targetID, userID)
	} else {
		return errors.New("会话不存在")
	}
	if err != nil {
		return err
	}
	defer rows.Close()
	var latest conversationReadCursor
	found := false
	for rows.Next() {
		var id, sentAtValue string
		if err := rows.Scan(&id, &sentAtValue); err != nil {
			return err
		}
		sentAt, err := time.Parse(time.RFC3339Nano, sentAtValue)
		if err != nil {
			return err
		}
		candidate := conversationReadCursor{SentAt: sentAt, MessageID: id}
		if !found || cursorComesAfter(candidate, latest) {
			latest, found = candidate, true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !found {
		return nil
	}
	return setConversationReadTx(tx, userID, conversationKey, latest)
}

func resolveConversationTx(tx *sql.Tx, userID, clientKey string) (resolvedConversation, error) {
	clientKey = strings.TrimSpace(clientKey)
	if strings.HasPrefix(clientKey, "group:") {
		groupID := strings.TrimPrefix(clientKey, "group:")
		if groupID == "" {
			return resolvedConversation{}, errors.New("群聊不存在")
		}
		var historyFromValue string
		err := tx.QueryRow(`SELECT history_from FROM group_members WHERE group_id = ? AND user_id = ?`,
			groupID, userID).Scan(&historyFromValue)
		if errors.Is(err, sql.ErrNoRows) {
			return resolvedConversation{}, ErrGroupMember
		}
		if err != nil {
			return resolvedConversation{}, err
		}
		historyFrom, err := time.Parse(time.RFC3339Nano, historyFromValue)
		if err != nil {
			return resolvedConversation{}, err
		}
		return resolvedConversation{
			ClientKey: clientKey, StableKey: clientKey, GroupID: groupID, HistoryFrom: historyFrom,
		}, nil
	}
	if !strings.HasPrefix(clientKey, "dm:") {
		return resolvedConversation{}, errors.New("会话不存在")
	}
	name := strings.TrimSpace(strings.TrimPrefix(clientKey, "dm:"))
	if name == "" {
		return resolvedConversation{}, errors.New("会话不存在")
	}
	targetID, targetName := CocoID, CocoName
	if !strings.EqualFold(name, CocoName) {
		target, err := findUserByName(tx, name)
		if err != nil {
			return resolvedConversation{}, err
		}
		if target == nil || target.ID == userID {
			return resolvedConversation{}, ErrNotFound
		}
		targetID, targetName = target.ID, target.Name
	}
	return resolvedConversation{
		ClientKey: "dm:" + targetName, StableKey: "dm:" + targetID, TargetID: targetID,
	}, nil
}

func clearedTime(q queryer, userID, conversationKey string) (time.Time, error) {
	var value string
	err := q.QueryRow(`SELECT cleared_at FROM cleared_at WHERE user_id = ? AND conversation_key = ?`,
		userID, conversationKey).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, value)
}

func (s *Store) MarkConversationRead(userID, clientKey, messageID string) (string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if user, err := userByID(tx, userID); err != nil {
		return "", err
	} else if user == nil {
		return "", ErrNotFound
	}
	conversation, err := resolveConversationTx(tx, userID, clientKey)
	if err != nil {
		return "", err
	}
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return "", errors.New("消息不能为空")
	}
	message, err := scanMessage(tx.QueryRow(`SELECT id, kind, from_id, COALESCE(to_id, ''),
		COALESCE(group_id, ''), text, sticker, sent_at, delivered_at, recalled_at
		FROM messages WHERE id = ?`, messageID))
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("消息不存在")
	}
	if err != nil {
		return "", err
	}
	if conversation.GroupID != "" {
		messageGroupID := message.GroupID
		if messageGroupID == "" {
			messageGroupID = PublicGroupID
		}
		if message.Kind != "group" || messageGroupID != conversation.GroupID ||
			!message.SentAt.After(conversation.HistoryFrom) {
			return "", errors.New("消息不属于这个会话")
		}
	} else if message.Kind != "private" ||
		!((message.FromID == userID && message.ToID == conversation.TargetID) ||
			(message.FromID == conversation.TargetID && message.ToID == userID)) {
		return "", errors.New("消息不属于这个会话")
	}
	clearedAt, err := clearedTime(tx, userID, conversation.StableKey)
	if err != nil {
		return "", err
	}
	if !message.SentAt.After(clearedAt) {
		return "", errors.New("消息已不在当前会话记录中")
	}
	if err := setConversationReadTx(tx, userID, conversation.StableKey, conversationReadCursor{
		SentAt: message.SentAt, MessageID: message.ID,
	}); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return conversation.ClientKey, nil
}
