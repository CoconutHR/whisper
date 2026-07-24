package chat

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrGroupNotFound     = errors.New("群聊不存在")
	ErrGroupForbidden    = errors.New("没有权限修改这个群聊")
	ErrGroupMember       = errors.New("不是这个群聊的成员")
	ErrSystemGroup       = errors.New("公共大厅不能修改或解散")
	ErrGroupNameTaken    = errors.New("群聊名称已被使用")
	ErrGroupNameReserved = errors.New("公共大厅是系统保留群名")
)

type databaseGroup struct {
	ID             string
	Name           string
	Signature      string
	OwnerID        string
	HistoryVisible bool
	System         bool
	CreatedAt      time.Time
}

func GroupConversationKey(groupID string) string {
	return "group:" + groupID
}

func ValidateGroupName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("群聊名称不能为空")
	}
	if utf8.RuneCountInString(name) > MaxGroupName {
		return fmt.Errorf("群聊名称不能超过%d个字符", MaxGroupName)
	}
	if name == PublicGroupName {
		return ErrGroupNameReserved
	}
	return nil
}

func normalizeGroupName(name string) string {
	return strings.TrimSpace(name)
}

func validateGroupSignature(signature string) error {
	if utf8.RuneCountInString(signature) > 240 {
		return errors.New("群个签不能超过240个字符")
	}
	return nil
}

func groupFromRow(scanner rowScanner) (*databaseGroup, error) {
	var group databaseGroup
	var historyVisible, system int
	var ownerID sql.NullString
	var createdAt string
	if err := scanner.Scan(&group.ID, &group.Name, &group.Signature, &ownerID,
		&historyVisible, &system, &createdAt); err != nil {
		return nil, err
	}
	if ownerID.Valid {
		group.OwnerID = ownerID.String
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil, err
	}
	group.CreatedAt = parsed
	group.HistoryVisible = historyVisible != 0
	group.System = system != 0
	return &group, nil
}

func groupByID(q queryer, groupID string) (*databaseGroup, error) {
	row := q.QueryRow(`SELECT id, name, signature, owner_id, history_visible, system, created_at
		FROM groups WHERE id = ?`, groupID)
	group, err := groupFromRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return group, err
}

func groupMemberIDsTx(tx *sql.Tx, groupID string) ([]string, error) {
	rows, err := tx.Query(`SELECT user_id FROM group_members WHERE group_id = ? ORDER BY user_id`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func groupViewTx(tx *sql.Tx, groupID, viewerID string, online map[string]bool) (GroupView, error) {
	group, err := groupByID(tx, groupID)
	if err != nil {
		return GroupView{}, err
	}
	if group == nil {
		return GroupView{}, ErrGroupNotFound
	}
	var memberCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM group_members WHERE group_id = ? AND user_id = ?`, groupID, viewerID).Scan(&memberCount); err != nil {
		return GroupView{}, err
	}
	if memberCount == 0 {
		return GroupView{}, ErrGroupMember
	}
	ownerName := ""
	if group.OwnerID != "" {
		if err := tx.QueryRow(`SELECT name FROM users WHERE id = ?`, group.OwnerID).Scan(&ownerName); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return GroupView{}, err
		}
	}
	rows, err := tx.Query(`SELECT u.name, u.id, gm.user_id
		FROM group_members gm JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = ? ORDER BY u.name`, groupID)
	if err != nil {
		return GroupView{}, err
	}
	defer rows.Close()
	members := []GroupMemberView{}
	for rows.Next() {
		var name, id, memberID string
		if err := rows.Scan(&name, &id, &memberID); err != nil {
			return GroupView{}, err
		}
		members = append(members, GroupMemberView{Name: name, Online: online[id], Owner: id == group.OwnerID})
	}
	if err := rows.Err(); err != nil {
		return GroupView{}, err
	}
	return GroupView{
		ID: group.ID, Name: group.Name, Signature: group.Signature, Owner: ownerName,
		IsOwner: group.OwnerID == viewerID, System: group.System,
		HistoryVisible: group.HistoryVisible, Members: members,
	}, nil
}

func groupsForUserTx(tx *sql.Tx, userID string, online map[string]bool) ([]GroupView, map[string]groupAccess, error) {
	rows, err := tx.Query(`SELECT g.id FROM groups g JOIN group_members gm ON gm.group_id = g.id
		WHERE gm.user_id = ? ORDER BY g.system DESC, g.name, g.id`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	groups := []GroupView{}
	access := map[string]groupAccess{}
	for rows.Next() {
		var groupID string
		if err := rows.Scan(&groupID); err != nil {
			return nil, nil, err
		}
		view, err := groupViewTx(tx, groupID, userID, online)
		if err != nil {
			return nil, nil, err
		}
		var historyFrom string
		if err := tx.QueryRow(`SELECT history_from FROM group_members WHERE group_id = ? AND user_id = ?`, groupID, userID).Scan(&historyFrom); err != nil {
			return nil, nil, err
		}
		parsed, err := time.Parse(time.RFC3339Nano, historyFrom)
		if err != nil {
			return nil, nil, err
		}
		groups = append(groups, view)
		access[groupID] = groupAccess{HistoryFrom: parsed}
	}
	return groups, access, rows.Err()
}

func resolveGroupMembers(tx *sql.Tx, names []string, ownerID string) (map[string]string, error) {
	ids := map[string]string{ownerID: ownerID}
	seenNames := map[string]bool{}
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		user, err := findUserByName(tx, name)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, ErrNotFound
		}
		if user.ID == ownerID {
			continue
		}
		if seenNames[strings.ToLower(user.Name)] {
			continue
		}
		seenNames[strings.ToLower(user.Name)] = true
		ids[user.ID] = user.ID
	}
	return ids, nil
}

func insertGroupSystemMessageTx(tx *sql.Tx, groupID, text string, sentAt time.Time) error {
	_, err := tx.Exec(`INSERT INTO messages(id, kind, from_id, to_id, group_id, text, sent_at, delivered_at)
		VALUES (?, 'group', '*', NULL, ?, ?, ?, ?)`, randomID(), groupID, text,
		sentAt.Format(time.RFC3339Nano), sentAt.Format(time.RFC3339Nano))
	return err
}

func (s *Store) CreateGroup(ownerID, name, signature string, historyVisible bool, memberNames []string) (GroupMutation, GroupView, error) {
	name = normalizeGroupName(name)
	if err := ValidateGroupName(name); err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	signature = strings.TrimSpace(signature)
	if err := validateGroupSignature(signature); err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	defer tx.Rollback()
	owner, err := userByID(tx, ownerID)
	if err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	if owner == nil {
		return GroupMutation{}, GroupView{}, ErrNotFound
	}
	members, err := resolveGroupMembers(tx, memberNames, ownerID)
	if err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	if len(members) < 2 {
		return GroupMutation{}, GroupView{}, errors.New("新建群聊至少需要选择一名成员")
	}
	groupID := randomID()
	now := time.Now()
	if _, err := tx.Exec(`INSERT INTO groups(id, name, signature, owner_id, history_visible, system, created_at)
		VALUES (?, ?, ?, ?, ?, 0, ?)`, groupID, name, signature, ownerID, boolInt(historyVisible), now.Format(time.RFC3339Nano)); err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	ids := make([]string, 0, len(members))
	for id := range members {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		historyFrom := now
		if historyVisible {
			historyFrom = now
		}
		if _, err := tx.Exec(`INSERT INTO group_members(group_id, user_id, history_from) VALUES (?, ?, ?)`, groupID, id, historyFrom.Format(time.RFC3339Nano)); err != nil {
			return GroupMutation{}, GroupView{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	view, err := s.GroupViewForUser(groupID, ownerID, nil)
	if err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	return GroupMutation{GroupID: groupID, MemberIDs: ids}, view, nil
}

func (s *Store) UpdateGroup(ownerID, groupID, name, signature string, historyVisible bool, memberNames []string) (GroupMutation, GroupView, error) {
	name = normalizeGroupName(name)
	if err := ValidateGroupName(name); err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	signature = strings.TrimSpace(signature)
	if err := validateGroupSignature(signature); err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	defer tx.Rollback()
	group, err := groupByID(tx, groupID)
	if err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	if group == nil {
		return GroupMutation{}, GroupView{}, ErrGroupNotFound
	}
	if group.System {
		return GroupMutation{}, GroupView{}, ErrSystemGroup
	}
	if group.OwnerID != ownerID {
		return GroupMutation{}, GroupView{}, ErrGroupForbidden
	}
	oldIDs, err := groupMemberIDsTx(tx, groupID)
	if err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	if memberNames == nil {
		memberNames = []string{}
		for _, id := range oldIDs {
			user, err := userByID(tx, id)
			if err != nil {
				return GroupMutation{}, GroupView{}, err
			}
			if user != nil && id != ownerID {
				memberNames = append(memberNames, user.Name)
			}
		}
	}
	members, err := resolveGroupMembers(tx, memberNames, ownerID)
	if err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	if len(members) < 2 {
		return GroupMutation{}, GroupView{}, errors.New("群聊至少需要保留一名成员")
	}
	now := time.Now()
	if _, err := tx.Exec(`UPDATE groups SET name = ?, signature = ?, history_visible = ? WHERE id = ?`,
		name, signature, boolInt(historyVisible), groupID); err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	oldSet := map[string]bool{}
	for _, id := range oldIDs {
		oldSet[id] = true
	}
	newIDs := make([]string, 0, len(members))
	for id := range members {
		newIDs = append(newIDs, id)
	}
	sort.Strings(newIDs)
	removed := []string{}
	for _, id := range oldIDs {
		_, keep := members[id]
		if !keep && id != ownerID {
			removed = append(removed, id)
			if _, err := tx.Exec(`DELETE FROM group_members WHERE group_id = ? AND user_id = ?`, groupID, id); err != nil {
				return GroupMutation{}, GroupView{}, err
			}
		}
	}
	for _, id := range newIDs {
		if oldSet[id] {
			continue
		}
		historyFrom := now
		if historyVisible {
			historyFrom = group.CreatedAt
		}
		if _, err := tx.Exec(`INSERT INTO group_members(group_id, user_id, history_from) VALUES (?, ?, ?)`, groupID, id, historyFrom.Format(time.RFC3339Nano)); err != nil {
			return GroupMutation{}, GroupView{}, err
		}
		newUser, err := userByID(tx, id)
		if err != nil {
			return GroupMutation{}, GroupView{}, err
		}
		if newUser != nil {
			if err := insertGroupSystemMessageTx(tx, groupID, fmt.Sprintf("%s 加入了群聊。", newUser.Name), now); err != nil {
				return GroupMutation{}, GroupView{}, err
			}
		}
		if err := seedLatestConversationReadTx(tx, id, GroupConversationKey(groupID)); err != nil {
			return GroupMutation{}, GroupView{}, err
		}
	}
	for _, id := range removed {
		if _, err := tx.Exec(`DELETE FROM conversation_reads WHERE user_id = ? AND conversation_key = ?`,
			id, GroupConversationKey(groupID)); err != nil {
			return GroupMutation{}, GroupView{}, err
		}
		removedUser, err := userByID(tx, id)
		if err != nil {
			return GroupMutation{}, GroupView{}, err
		}
		if removedUser != nil {
			if err := insertGroupSystemMessageTx(tx, groupID, fmt.Sprintf("%s 已被移出群聊。", removedUser.Name), now); err != nil {
				return GroupMutation{}, GroupView{}, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	view, err := s.GroupViewForUser(groupID, ownerID, nil)
	if err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	return GroupMutation{GroupID: groupID, MemberIDs: newIDs, RemovedIDs: removed}, view, nil
}

func (s *Store) TransferGroup(ownerID, groupID, newOwnerName string) (GroupMutation, GroupView, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	defer tx.Rollback()
	group, err := groupByID(tx, groupID)
	if err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	if group == nil {
		return GroupMutation{}, GroupView{}, ErrGroupNotFound
	}
	if group.System {
		return GroupMutation{}, GroupView{}, ErrSystemGroup
	}
	if group.OwnerID != ownerID {
		return GroupMutation{}, GroupView{}, ErrGroupForbidden
	}
	newOwner, err := findUserByName(tx, newOwnerName)
	if err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	if newOwner == nil {
		return GroupMutation{}, GroupView{}, ErrNotFound
	}
	var memberCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM group_members WHERE group_id = ? AND user_id = ?`, groupID, newOwner.ID).Scan(&memberCount); err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	if memberCount == 0 {
		return GroupMutation{}, GroupView{}, errors.New("新群主必须是群成员")
	}
	if _, err := tx.Exec(`UPDATE groups SET owner_id = ? WHERE id = ?`, newOwner.ID, groupID); err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	oldOwner, err := userByID(tx, ownerID)
	if err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	now := time.Now()
	if err := insertGroupSystemMessageTx(tx, groupID, fmt.Sprintf("%s 将群主转移给了 %s。", oldOwner.Name, newOwner.Name), now); err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	ids, err := groupMemberIDsTx(tx, groupID)
	if err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	if err := tx.Commit(); err != nil {
		return GroupMutation{}, GroupView{}, err
	}
	view, err := s.GroupViewForUser(groupID, ownerID, nil)
	return GroupMutation{GroupID: groupID, MemberIDs: ids}, view, err
}

func (s *Store) LeaveGroup(userID, groupID string) (GroupMutation, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return GroupMutation{}, err
	}
	defer tx.Rollback()
	group, err := groupByID(tx, groupID)
	if err != nil {
		return GroupMutation{}, err
	}
	if group == nil {
		return GroupMutation{}, ErrGroupNotFound
	}
	if group.System {
		return GroupMutation{}, ErrSystemGroup
	}
	if group.OwnerID == userID {
		return GroupMutation{}, errors.New("群主不能直接退出，请先转移群主或解散群聊")
	}
	var memberCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM group_members WHERE group_id = ? AND user_id = ?`, groupID, userID).Scan(&memberCount); err != nil {
		return GroupMutation{}, err
	}
	if memberCount == 0 {
		return GroupMutation{}, ErrGroupMember
	}
	user, err := userByID(tx, userID)
	if err != nil {
		return GroupMutation{}, err
	}
	if _, err := tx.Exec(`DELETE FROM group_members WHERE group_id = ? AND user_id = ?`, groupID, userID); err != nil {
		return GroupMutation{}, err
	}
	if _, err := tx.Exec(`DELETE FROM conversation_reads WHERE user_id = ? AND conversation_key = ?`,
		userID, GroupConversationKey(groupID)); err != nil {
		return GroupMutation{}, err
	}
	if err := insertGroupSystemMessageTx(tx, groupID, fmt.Sprintf("%s 退出了群聊。", user.Name), time.Now()); err != nil {
		return GroupMutation{}, err
	}
	ids, err := groupMemberIDsTx(tx, groupID)
	if err != nil {
		return GroupMutation{}, err
	}
	if err := tx.Commit(); err != nil {
		return GroupMutation{}, err
	}
	return GroupMutation{GroupID: groupID, MemberIDs: ids, RemovedIDs: []string{userID}}, nil
}

func (s *Store) DissolveGroup(ownerID, groupID string) (GroupMutation, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return GroupMutation{}, err
	}
	defer tx.Rollback()
	group, err := groupByID(tx, groupID)
	if err != nil {
		return GroupMutation{}, err
	}
	if group == nil {
		return GroupMutation{}, ErrGroupNotFound
	}
	if group.System {
		return GroupMutation{}, ErrSystemGroup
	}
	if group.OwnerID != ownerID {
		return GroupMutation{}, ErrGroupForbidden
	}
	ids, err := groupMemberIDsTx(tx, groupID)
	if err != nil {
		return GroupMutation{}, err
	}
	if _, err := tx.Exec(`DELETE FROM messages WHERE group_id = ?`, groupID); err != nil {
		return GroupMutation{}, err
	}
	if _, err := tx.Exec(`DELETE FROM cleared_at WHERE conversation_key = ?`, GroupConversationKey(groupID)); err != nil {
		return GroupMutation{}, err
	}
	if _, err := tx.Exec(`DELETE FROM conversation_reads WHERE conversation_key = ?`, GroupConversationKey(groupID)); err != nil {
		return GroupMutation{}, err
	}
	if _, err := tx.Exec(`DELETE FROM groups WHERE id = ?`, groupID); err != nil {
		return GroupMutation{}, err
	}
	if err := tx.Commit(); err != nil {
		return GroupMutation{}, err
	}
	return GroupMutation{GroupID: groupID, RemovedIDs: ids}, nil
}

func (s *Store) GroupViewForUser(groupID, userID string, online map[string]bool) (GroupView, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return GroupView{}, err
	}
	defer tx.Rollback()
	if online == nil {
		online = map[string]bool{}
	}
	return groupViewTx(tx, groupID, userID, online)
}

func (s *Store) SendGroupMessage(fromID, groupID, text string) (*Message, []string, error) {
	return s.SendGroupMessageContent(fromID, groupID, MessageContent{Text: text})
}

func (s *Store) SendGroupMessageContent(fromID, groupID string, content MessageContent) (*Message, []string, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		groupID = PublicGroupID
	}
	content, err := normalizeMessageContent(content)
	if err != nil {
		return nil, nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	group, err := groupByID(tx, groupID)
	if err != nil {
		return nil, nil, err
	}
	if group == nil {
		return nil, nil, ErrGroupNotFound
	}
	var memberCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM group_members WHERE group_id = ? AND user_id = ?`, groupID, fromID).Scan(&memberCount); err != nil {
		return nil, nil, err
	}
	if memberCount == 0 {
		return nil, nil, ErrGroupMember
	}
	now := time.Now()
	message := &Message{
		ID: randomID(), Kind: "group", FromID: fromID, GroupID: groupID,
		Text: content.Text, Sticker: content.Sticker, SentAt: now,
	}
	if _, err := tx.Exec(`INSERT INTO messages(id, kind, from_id, to_id, group_id, text, sticker, sent_at, delivered_at)
		VALUES (?, 'group', ?, NULL, ?, ?, ?, ?, NULL)`, message.ID, message.FromID, groupID,
		message.Text, message.Sticker, now.Format(time.RFC3339Nano)); err != nil {
		return nil, nil, err
	}
	if content.ForwardAttachmentID != "" {
		message.Attachments, err = attachForwardedFileTx(tx, message.ID, fromID, content.ForwardAttachmentID)
	} else {
		message.Attachments, err = attachMessageFilesTx(tx, message.ID, fromID, content.AttachmentIDs)
	}
	if err != nil {
		return nil, nil, err
	}
	if content.StickerAttachmentID != "" {
		message.StickerAttachment, err = attachStickerTx(tx, message.ID, fromID, content.StickerAttachmentID)
		if err != nil {
			return nil, nil, err
		}
	}
	ids, err := groupMemberIDsTx(tx, groupID)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	return message, ids, nil
}
