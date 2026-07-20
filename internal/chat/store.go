package chat

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

const (
	CocoID          = "coco"
	CocoName        = "coco"
	PublicGroupID   = "public"
	PublicGroupName = "公共大厅"
	MaxNameRunes    = 7
	MaxMessage      = 2000
	MaxGroupName    = 32
	schemaVersion   = 2
)

var (
	ErrUnauthorized = errors.New("用户名或密码错误")
	ErrNotFound     = errors.New("用户不存在")
	ErrNameTaken    = errors.New("该名称已被使用")
)

type Settings struct {
	ShowMessageTime bool   `json:"showMessageTime"`
	ParseLatex      bool   `json:"parseLatex"`
	Theme           string `json:"theme"`
}

type Message struct {
	ID          string     `json:"id"`
	Kind        string     `json:"kind"`
	FromID      string     `json:"fromId"`
	ToID        string     `json:"toId,omitempty"`
	Text        string     `json:"text"`
	SentAt      time.Time  `json:"sentAt"`
	DeliveredAt *time.Time `json:"deliveredAt,omitempty"`
	GroupID     string     `json:"groupId,omitempty"`
}

type SelfView struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Signature string   `json:"signature"`
	Settings  Settings `json:"settings"`
}

type MemberView struct {
	Name      string `json:"name"`
	Online    bool   `json:"online"`
	Reserved  bool   `json:"reserved,omitempty"`
	Signature string `json:"signature,omitempty"`
}

type MessageView struct {
	ID       string `json:"id"`
	From     string `json:"from"`
	Text     string `json:"text"`
	SentAt   string `json:"sentAt"`
	Delivery string `json:"delivery"`
	System   bool   `json:"system,omitempty"`
}

type GroupMemberView struct {
	Name   string `json:"name"`
	Online bool   `json:"online"`
	Owner  bool   `json:"owner,omitempty"`
}

type GroupView struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Signature      string            `json:"signature"`
	Owner          string            `json:"owner"`
	IsOwner        bool              `json:"isOwner"`
	System         bool              `json:"system"`
	HistoryVisible bool              `json:"historyVisible"`
	Members        []GroupMemberView `json:"members"`
}

type GroupMutation struct {
	GroupID    string
	MemberIDs  []string
	RemovedIDs []string
}

type Bootstrap struct {
	Self          SelfView                 `json:"self"`
	Members       []MemberView             `json:"members"`
	Friends       []string                 `json:"friends"`
	FriendColors  map[string]string        `json:"friendColors"`
	Groups        []GroupView              `json:"groups"`
	Conversations map[string][]MessageView `json:"conversations"`
}

type DeliveryNotice struct {
	MessageID string
	SenderID  string
}

type StoreConfig struct {
	DatabasePath   string
	UserBackupPath string
}

type Store struct {
	db                 *sql.DB
	databasePath       string
	userBackupPath     string
	backupMu           sync.Mutex
	plaintextPasswords map[string]string
}

type databaseUser struct {
	ID           string
	Name         string
	PasswordHash string
	Signature    string
	Settings     Settings
	CreatedAt    time.Time
}

type groupAccess struct {
	HistoryFrom time.Time
}

type plaintextBackup struct {
	Version   int                   `json:"version"`
	Warning   string                `json:"warning"`
	UpdatedAt string                `json:"updatedAt"`
	Users     []plaintextBackupUser `json:"users"`
}

type plaintextBackupUser struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Password      string            `json:"password"`
	PasswordKnown bool              `json:"passwordKnown"`
	PasswordHash  string            `json:"passwordHash"`
	Signature     string            `json:"signature"`
	Settings      Settings          `json:"settings"`
	Friends       []string          `json:"friends"`
	FriendIDs     []string          `json:"friendIds"`
	FriendColors  map[string]string `json:"friendColors"`
	ClearedAt     map[string]string `json:"clearedAt"`
	CreatedAt     string            `json:"createdAt"`
}

type queryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type rowScanner interface {
	Scan(dest ...any) error
}

func NewStore(config StoreConfig) (*Store, error) {
	if strings.TrimSpace(config.DatabasePath) == "" {
		return nil, errors.New("SQLite 数据库路径不能为空")
	}
	databasePath, err := filepath.Abs(config.DatabasePath)
	if err != nil {
		return nil, err
	}
	if config.UserBackupPath == "" {
		config.UserBackupPath = filepath.Join(filepath.Dir(databasePath), "users-backup.json")
	}
	userBackupPath, err := filepath.Abs(config.UserBackupPath)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(databasePath), 0o700); err != nil {
		return nil, err
	}
	dsnURL := &url.URL{Scheme: "file", Path: databasePath}
	db, err := sql.Open("sqlite", dsnURL.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &Store{
		db: db, databasePath: databasePath,
		userBackupPath: userBackupPath, plaintextPasswords: map[string]string{},
	}
	if err := store.initializeSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := os.Chmod(databasePath, 0o600); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.loadPlaintextPasswords(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.syncUserBackup("", "", false); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) DatabasePath() string {
	return s.databasePath
}

func (s *Store) UserBackupPath() string {
	return s.userBackupPath
}

func (s *Store) initializeSchema() error {
	statements := []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA journal_mode = WAL`,
		`PRAGMA busy_timeout = 5000`,
		`CREATE TABLE IF NOT EXISTS meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL COLLATE NOCASE UNIQUE,
			password_hash TEXT NOT NULL,
			signature TEXT NOT NULL DEFAULT '',
			show_message_time INTEGER NOT NULL DEFAULT 1,
			parse_latex INTEGER NOT NULL DEFAULT 1,
			theme TEXT NOT NULL DEFAULT 'dune',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS friends (
			user_id TEXT NOT NULL,
			friend_id TEXT NOT NULL,
			PRIMARY KEY (user_id, friend_id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS friend_colors (
			user_id TEXT NOT NULL,
			friend_id TEXT NOT NULL,
			color TEXT NOT NULL,
			PRIMARY KEY (user_id, friend_id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			signature TEXT NOT NULL DEFAULT '',
			owner_id TEXT,
			history_visible INTEGER NOT NULL DEFAULT 0,
			system INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL
		)`,
		`CREATE TABLE IF NOT EXISTS group_members (
			group_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			history_from TEXT NOT NULL,
			PRIMARY KEY (group_id, user_id),
			FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS group_members_user_idx ON group_members(user_id, group_id)`,
		`CREATE TABLE IF NOT EXISTS cleared_at (
			user_id TEXT NOT NULL,
			conversation_key TEXT NOT NULL,
			cleared_at TEXT NOT NULL,
			PRIMARY KEY (user_id, conversation_key),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL CHECK (kind IN ('group', 'private')),
			from_id TEXT NOT NULL,
			to_id TEXT,
			group_id TEXT,
			text TEXT NOT NULL,
			sent_at TEXT NOT NULL,
			delivered_at TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS messages_sent_at_idx ON messages(sent_at, id)`,
		`CREATE INDEX IF NOT EXISTS messages_recipient_idx ON messages(to_id, delivered_at)`,
		`INSERT INTO meta(key, value) VALUES ('schema_version', '2')
			ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("初始化 SQLite: %w", err)
		}
	}
	return s.migrateGroups()
}

func (s *Store) migrateGroups() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	columns, err := tx.Query(`PRAGMA table_info(messages)`)
	if err != nil {
		return err
	}
	hasGroupID := false
	for columns.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue sql.NullString
		if err := columns.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			columns.Close()
			return err
		}
		if name == "group_id" {
			hasGroupID = true
		}
	}
	if err := columns.Close(); err != nil {
		return err
	}
	if !hasGroupID {
		if _, err := tx.Exec(`ALTER TABLE messages ADD COLUMN group_id TEXT`); err != nil {
			return err
		}
	}
	now := time.Now().Format(time.RFC3339Nano)
	if _, err := tx.Exec(`INSERT OR IGNORE INTO groups(id, name, signature, owner_id, history_visible, system, created_at)
		VALUES (?, ?, '', NULL, 0, 1, ?)`, PublicGroupID, PublicGroupName, now); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO group_members(group_id, user_id, history_from)
		SELECT ?, id, '0001-01-01T00:00:00Z' FROM users`, PublicGroupID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE group_members SET history_from = '0001-01-01T00:00:00Z' WHERE group_id = ?`, PublicGroupID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE messages SET group_id = ? WHERE kind = 'group' AND (group_id IS NULL OR group_id = '')`, PublicGroupID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO cleared_at(user_id, conversation_key, cleared_at)
		SELECT user_id, ?, cleared_at FROM cleared_at WHERE conversation_key = 'group'
		ON CONFLICT(user_id, conversation_key) DO UPDATE SET cleared_at = excluded.cleared_at`, GroupConversationKey(PublicGroupID)); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM cleared_at WHERE conversation_key = 'group'`); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func DefaultSettings() Settings {
	return Settings{ShowMessageTime: true, ParseLatex: true, Theme: "dune"}
}

func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("个人名称不能为空")
	}
	if utf8.RuneCountInString(name) > MaxNameRunes {
		return fmt.Errorf("个人名称不能超过%d个字符", MaxNameRunes)
	}
	if strings.EqualFold(name, CocoName) {
		return errors.New("coco 是系统保留名称")
	}
	return nil
}

func (s *Store) Register(name, password string) (SelfView, error) {
	name = strings.TrimSpace(name)
	if err := ValidateName(name); err != nil {
		return SelfView{}, err
	}
	if password == "" {
		return SelfView{}, errors.New("密码不能为空")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return SelfView{}, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return SelfView{}, err
	}
	defer tx.Rollback()
	existing, err := findUserByName(tx, name)
	if err != nil {
		return SelfView{}, err
	}
	if existing != nil {
		return SelfView{}, ErrNameTaken
	}

	now := time.Now()
	user := &databaseUser{
		ID: randomID(), Name: name, PasswordHash: string(hash), Settings: DefaultSettings(), CreatedAt: now,
	}
	if _, err := tx.Exec(`INSERT INTO users(
		id, name, password_hash, signature, show_message_time, parse_latex, theme, created_at
	) VALUES (?, ?, ?, '', 1, 1, 'dune', ?)`, user.ID, user.Name, user.PasswordHash, now.Format(time.RFC3339Nano)); err != nil {
		if isUniqueError(err) {
			return SelfView{}, ErrNameTaken
		}
		return SelfView{}, err
	}
	if _, err := tx.Exec(`INSERT INTO friends(user_id, friend_id) VALUES (?, ?)`, user.ID, CocoID); err != nil {
		return SelfView{}, err
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO group_members(group_id, user_id, history_from) VALUES (?, ?, ?)`,
		PublicGroupID, user.ID, "0001-01-01T00:00:00Z"); err != nil {
		return SelfView{}, err
	}
	deliveredAt := now.Format(time.RFC3339Nano)
	if _, err := tx.Exec(`INSERT INTO messages(id, kind, from_id, to_id, text, sent_at, delivered_at)
		VALUES (?, 'private', ?, ?, ?, ?, ?)`, randomID(), CocoID, user.ID,
		"这里的消息仅你自己可见。", now.Format(time.RFC3339Nano), deliveredAt); err != nil {
		return SelfView{}, err
	}
	if err := tx.Commit(); err != nil {
		return SelfView{}, err
	}
	if err := s.syncUserBackup(user.ID, password, true); err != nil {
		return SelfView{}, err
	}
	return selfView(user), nil
}

func (s *Store) Authenticate(name, password string) (SelfView, error) {
	user, err := findUserByName(s.db, strings.TrimSpace(name))
	if err != nil {
		return SelfView{}, err
	}
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return SelfView{}, ErrUnauthorized
	}
	if err := s.syncUserBackup(user.ID, password, true); err != nil {
		return SelfView{}, err
	}
	return selfView(user), nil
}

func (s *Store) User(id string) (SelfView, error) {
	user, err := userByID(s.db, id)
	if err != nil {
		return SelfView{}, err
	}
	if user == nil {
		return SelfView{}, ErrNotFound
	}
	return selfView(user), nil
}

func (s *Store) Bootstrap(userID string, online map[string]bool) (Bootstrap, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Bootstrap{}, err
	}
	defer tx.Rollback()
	viewer, err := userByID(tx, userID)
	if err != nil {
		return Bootstrap{}, err
	}
	if viewer == nil {
		return Bootstrap{}, ErrNotFound
	}
	users, err := allUsers(tx)
	if err != nil {
		return Bootstrap{}, err
	}
	usersByID := make(map[string]*databaseUser, len(users))
	for _, user := range users {
		usersByID[user.ID] = user
	}

	groups, groupAccess, err := groupsForUserTx(tx, userID, online)
	if err != nil {
		return Bootstrap{}, err
	}
	conversations := map[string][]MessageView{"dm:coco": {}}
	for _, group := range groups {
		conversations[GroupConversationKey(group.ID)] = []MessageView{}
	}
	result := Bootstrap{
		Self:    selfView(viewer),
		Members: []MemberView{{Name: CocoName, Online: true, Reserved: true, Signature: "仅自己可见"}},
		Friends: []string{}, FriendColors: map[string]string{}, Groups: groups,
		Conversations: conversations,
	}
	otherUsers := make([]*databaseUser, 0, len(users)-1)
	for _, user := range users {
		if user.ID != userID {
			otherUsers = append(otherUsers, user)
		}
	}
	sort.Slice(otherUsers, func(i, j int) bool { return otherUsers[i].Name < otherUsers[j].Name })
	for _, user := range otherUsers {
		result.Members = append(result.Members, MemberView{
			Name: user.Name, Online: online[user.ID], Signature: user.Signature,
		})
		result.Conversations["dm:"+user.Name] = []MessageView{}
	}

	friendRows, err := tx.Query(`SELECT f.friend_id, COALESCE(c.color, '')
		FROM friends f LEFT JOIN friend_colors c
		ON c.user_id = f.user_id AND c.friend_id = f.friend_id
		WHERE f.user_id = ?`, userID)
	if err != nil {
		return Bootstrap{}, err
	}
	for friendRows.Next() {
		var friendID, color string
		if err := friendRows.Scan(&friendID, &color); err != nil {
			friendRows.Close()
			return Bootstrap{}, err
		}
		friendName := CocoName
		if friendID != CocoID {
			friend := usersByID[friendID]
			if friend == nil {
				continue
			}
			friendName = friend.Name
		}
		result.Friends = append(result.Friends, friendName)
		if color != "" {
			result.FriendColors[friendName] = color
		}
	}
	if err := friendRows.Close(); err != nil {
		return Bootstrap{}, err
	}
	sort.Strings(result.Friends)

	clearedAt, err := clearedTimes(tx, userID)
	if err != nil {
		return Bootstrap{}, err
	}
	messageRows, err := tx.Query(`SELECT id, kind, from_id, COALESCE(to_id, ''), COALESCE(group_id, ''), text, sent_at, delivered_at
		FROM messages WHERE (kind = 'group' AND (group_id IN (SELECT group_id FROM group_members WHERE user_id = ?) OR group_id IS NULL))
		OR from_id = ? OR to_id = ? ORDER BY sent_at, id`, userID, userID, userID)
	if err != nil {
		return Bootstrap{}, err
	}
	for messageRows.Next() {
		message, err := scanMessage(messageRows)
		if err != nil {
			messageRows.Close()
			return Bootstrap{}, err
		}
		key, visible := visibleConversation(userID, message, clearedAt, usersByID, groupAccess)
		if !visible {
			continue
		}
		result.Conversations[key] = append(result.Conversations[key], messageView(userID, message, usersByID))
	}
	if err := messageRows.Close(); err != nil {
		return Bootstrap{}, err
	}
	if err := tx.Commit(); err != nil {
		return Bootstrap{}, err
	}
	return result, nil
}

func (s *Store) UpdateProfile(userID, name, signature string) (SelfView, string, error) {
	name = strings.TrimSpace(name)
	if err := ValidateName(name); err != nil {
		return SelfView{}, "", err
	}
	if utf8.RuneCountInString(signature) > 240 {
		return SelfView{}, "", errors.New("个性签名不能超过240个字符")
	}
	signature = strings.TrimSpace(signature)

	tx, err := s.db.Begin()
	if err != nil {
		return SelfView{}, "", err
	}
	defer tx.Rollback()
	user, err := userByID(tx, userID)
	if err != nil {
		return SelfView{}, "", err
	}
	if user == nil {
		return SelfView{}, "", ErrNotFound
	}
	existing, err := findUserByName(tx, name)
	if err != nil {
		return SelfView{}, "", err
	}
	if existing != nil && existing.ID != userID {
		return SelfView{}, "", ErrNameTaken
	}
	previousName := user.Name
	if _, err := tx.Exec(`UPDATE users SET name = ?, signature = ? WHERE id = ?`, name, signature, userID); err != nil {
		if isUniqueError(err) {
			return SelfView{}, "", ErrNameTaken
		}
		return SelfView{}, "", err
	}
	if err := tx.Commit(); err != nil {
		return SelfView{}, "", err
	}
	user.Name = name
	user.Signature = signature
	if err := s.syncUserBackup("", "", false); err != nil {
		return SelfView{}, "", err
	}
	return selfView(user), previousName, nil
}

func (s *Store) UpdatePassword(userID, currentPassword, newPassword string) error {
	if newPassword == "" {
		return errors.New("新密码不能为空")
	}
	user, err := userByID(s.db, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrNotFound
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)) != nil {
		return errors.New("当前密码不正确")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, string(hash), userID); err != nil {
		return err
	}
	return s.syncUserBackup(userID, newPassword, true)
}

func (s *Store) UpdateSettings(userID string, settings Settings) error {
	if settings.Theme != "dune" && settings.Theme != "ocean" && settings.Theme != "paper" {
		return errors.New("未知配色方案")
	}
	result, err := s.db.Exec(`UPDATE users SET show_message_time = ?, parse_latex = ?, theme = ? WHERE id = ?`,
		boolInt(settings.ShowMessageTime), boolInt(settings.ParseLatex), settings.Theme, userID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return ErrNotFound
	}
	return s.syncUserBackup("", "", false)
}

func (s *Store) SendMessage(fromID, scope, targetName, text string, targetOnline bool) (*Message, string, error) {
	if scope == "group" {
		groupID := strings.TrimSpace(targetName)
		if groupID == "" {
			groupID = PublicGroupID
		}
		message, _, err := s.SendGroupMessage(fromID, groupID, text)
		return message, "", err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, "", errors.New("消息不能为空")
	}
	if utf8.RuneCountInString(text) > MaxMessage {
		return nil, "", fmt.Errorf("消息不能超过%d个字符", MaxMessage)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback()
	sender, err := userByID(tx, fromID)
	if err != nil {
		return nil, "", err
	}
	if sender == nil {
		return nil, "", ErrNotFound
	}

	message := &Message{ID: randomID(), Kind: scope, FromID: fromID, Text: text, SentAt: time.Now()}
	targetID := ""
	friendChanged := false
	if scope == "private" {
		if strings.EqualFold(targetName, CocoName) {
			targetID = CocoID
			deliveredAt := message.SentAt
			message.DeliveredAt = &deliveredAt
		} else {
			target, err := findUserByName(tx, targetName)
			if err != nil {
				return nil, "", err
			}
			if target == nil {
				return nil, "", ErrNotFound
			}
			if target.ID == fromID {
				return nil, "", errors.New("不能向本人发送消息")
			}
			targetID = target.ID
			if _, err := tx.Exec(`INSERT OR IGNORE INTO friends(user_id, friend_id) VALUES (?, ?)`, targetID, fromID); err != nil {
				return nil, "", err
			}
			if targetOnline {
				deliveredAt := message.SentAt
				message.DeliveredAt = &deliveredAt
			}
		}
		message.ToID = targetID
		if _, err := tx.Exec(`INSERT OR IGNORE INTO friends(user_id, friend_id) VALUES (?, ?)`, fromID, targetID); err != nil {
			return nil, "", err
		}
		friendChanged = true
	} else {
		return nil, "", errors.New("未知消息类型")
	}
	var deliveredAt any
	if message.DeliveredAt != nil {
		deliveredAt = message.DeliveredAt.Format(time.RFC3339Nano)
	}
	if _, err := tx.Exec(`INSERT INTO messages(id, kind, from_id, to_id, text, sent_at, delivered_at)
		VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, ?)`, message.ID, message.Kind, message.FromID,
		message.ToID, message.Text, message.SentAt.Format(time.RFC3339Nano), deliveredAt); err != nil {
		return nil, "", err
	}
	if err := tx.Commit(); err != nil {
		return nil, "", err
	}
	if friendChanged {
		if err := s.syncUserBackup("", "", false); err != nil {
			return nil, "", err
		}
	}
	clone := *message
	return &clone, targetID, nil
}

func (s *Store) ClearConversation(userID, targetName string) error {
	if user, err := userByID(s.db, userID); err != nil {
		return err
	} else if user == nil {
		return ErrNotFound
	}
	key := GroupConversationKey(PublicGroupID)
	if targetName != "group" && targetName != key {
		if strings.HasPrefix(targetName, "group:") {
			key = targetName
		} else {
			if strings.EqualFold(targetName, CocoName) {
				key = "dm:" + CocoID
			} else {
				target, err := findUserByName(s.db, targetName)
				if err != nil {
					return err
				}
				if target == nil {
					return ErrNotFound
				}
				key = "dm:" + target.ID
			}
		}
	}
	if _, err := s.db.Exec(`INSERT INTO cleared_at(user_id, conversation_key, cleared_at) VALUES (?, ?, ?)
		ON CONFLICT(user_id, conversation_key) DO UPDATE SET cleared_at = excluded.cleared_at`,
		userID, key, time.Now().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return s.syncUserBackup("", "", false)
}

func (s *Store) DeleteFriend(userID, name string) error {
	if strings.EqualFold(name, CocoName) {
		return errors.New("coco 不能删除")
	}
	target, err := findUserByName(s.db, name)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotFound
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`DELETE FROM friends WHERE user_id = ? AND friend_id = ?`, userID, target.ID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		if user, err := userByID(tx, userID); err != nil {
			return err
		} else if user == nil {
			return ErrNotFound
		}
	}
	if _, err := tx.Exec(`DELETE FROM friend_colors WHERE user_id = ? AND friend_id = ?`, userID, target.ID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return s.syncUserBackup("", "", false)
}

func (s *Store) SetFriendColor(userID, name, color string) error {
	color = strings.ToLower(strings.TrimSpace(color))
	if color != "default" && color != "green" && color != "blue" &&
		color != "cyan" && color != "amber" && color != "rose" {
		return errors.New("未知的好友消息颜色")
	}
	targetID := CocoID
	if !strings.EqualFold(name, CocoName) {
		target, err := findUserByName(s.db, name)
		if err != nil {
			return err
		}
		if target == nil {
			return ErrNotFound
		}
		targetID = target.ID
	}
	var isFriend int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM friends WHERE user_id = ? AND friend_id = ?`, userID, targetID).Scan(&isFriend); err != nil {
		return err
	}
	if isFriend == 0 {
		return errors.New("只能设置好友的消息颜色")
	}
	if color == "default" {
		if _, err := s.db.Exec(`DELETE FROM friend_colors WHERE user_id = ? AND friend_id = ?`, userID, targetID); err != nil {
			return err
		}
	} else {
		if _, err := s.db.Exec(`INSERT INTO friend_colors(user_id, friend_id, color) VALUES (?, ?, ?)
			ON CONFLICT(user_id, friend_id) DO UPDATE SET color = excluded.color`, userID, targetID, color); err != nil {
			return err
		}
	}
	return s.syncUserBackup("", "", false)
}

func (s *Store) MarkDelivered(userID string) ([]DeliveryNotice, error) {
	if user, err := userByID(s.db, userID); err != nil {
		return nil, err
	} else if user == nil {
		return nil, ErrNotFound
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT id, from_id FROM messages WHERE to_id = ? AND delivered_at IS NULL`, userID)
	if err != nil {
		return nil, err
	}
	notices := []DeliveryNotice{}
	for rows.Next() {
		var notice DeliveryNotice
		if err := rows.Scan(&notice.MessageID, &notice.SenderID); err != nil {
			rows.Close()
			return nil, err
		}
		notices = append(notices, notice)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if len(notices) > 0 {
		if _, err := tx.Exec(`UPDATE messages SET delivered_at = ? WHERE to_id = ? AND delivered_at IS NULL`,
			time.Now().Format(time.RFC3339Nano), userID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return notices, nil
}

func (s *Store) MessageView(viewerID string, message *Message) MessageView {
	from := CocoName
	system := false
	if message.FromID == "*" {
		from = "*"
		system = true
	} else if message.FromID != CocoID {
		from = s.NameForID(message.FromID)
	}
	delivery := "sent"
	if message.FromID == viewerID && message.ToID != "" && message.DeliveredAt == nil {
		delivery = "queued"
	}
	return MessageView{
		ID: message.ID, From: from, Text: message.Text,
		SentAt: message.SentAt.Format(time.RFC3339Nano), Delivery: delivery, System: system,
	}
}

func (s *Store) NameForID(id string) string {
	if id == CocoID {
		return CocoName
	}
	user, err := userByID(s.db, id)
	if err != nil || user == nil {
		return ""
	}
	return user.Name
}

func (s *Store) IDForName(name string) string {
	if strings.EqualFold(name, CocoName) {
		return CocoID
	}
	user, err := findUserByName(s.db, name)
	if err != nil || user == nil {
		return ""
	}
	return user.ID
}

func (s *Store) loadPlaintextPasswords() error {
	data, err := os.ReadFile(s.userBackupPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var backup plaintextBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		return fmt.Errorf("读取明文用户备份: %w", err)
	}
	for _, user := range backup.Users {
		if user.PasswordKnown && user.Password != "" {
			s.plaintextPasswords[user.ID] = user.Password
		}
	}
	return nil
}

func (s *Store) syncUserBackup(userID, password string, passwordKnown bool) error {
	s.backupMu.Lock()
	defer s.backupMu.Unlock()
	if passwordKnown && userID != "" {
		s.plaintextPasswords[userID] = password
	}
	return s.writeUserBackupLocked()
}

func (s *Store) writeUserBackupLocked() error {
	users, err := allUsers(s.db)
	if err != nil {
		return err
	}
	usersByID := make(map[string]*databaseUser, len(users))
	recordsByID := make(map[string]*plaintextBackupUser, len(users))
	backup := plaintextBackup{
		Version:   1,
		Warning:   "此文件包含明文密码，仅用于本机备份；不得提交到 Git 或分享。",
		UpdatedAt: time.Now().Format(time.RFC3339Nano),
		Users:     make([]plaintextBackupUser, 0, len(users)),
	}
	for _, user := range users {
		usersByID[user.ID] = user
		password, known := s.plaintextPasswords[user.ID]
		record := plaintextBackupUser{
			ID: user.ID, Name: user.Name, Password: password, PasswordKnown: known,
			PasswordHash: user.PasswordHash, Signature: user.Signature, Settings: user.Settings,
			Friends: []string{}, FriendIDs: []string{}, FriendColors: map[string]string{},
			ClearedAt: map[string]string{}, CreatedAt: user.CreatedAt.Format(time.RFC3339Nano),
		}
		backup.Users = append(backup.Users, record)
		recordsByID[user.ID] = &backup.Users[len(backup.Users)-1]
	}

	friendRows, err := s.db.Query(`SELECT user_id, friend_id FROM friends ORDER BY user_id, friend_id`)
	if err != nil {
		return err
	}
	for friendRows.Next() {
		var userID, friendID string
		if err := friendRows.Scan(&userID, &friendID); err != nil {
			friendRows.Close()
			return err
		}
		record := recordsByID[userID]
		if record == nil {
			continue
		}
		record.FriendIDs = append(record.FriendIDs, friendID)
		friendName := CocoName
		if friendID != CocoID {
			friend := usersByID[friendID]
			if friend == nil {
				continue
			}
			friendName = friend.Name
		}
		record.Friends = append(record.Friends, friendName)
	}
	if err := friendRows.Close(); err != nil {
		return err
	}

	colorRows, err := s.db.Query(`SELECT user_id, friend_id, color FROM friend_colors ORDER BY user_id, friend_id`)
	if err != nil {
		return err
	}
	for colorRows.Next() {
		var userID, friendID, color string
		if err := colorRows.Scan(&userID, &friendID, &color); err != nil {
			colorRows.Close()
			return err
		}
		record := recordsByID[userID]
		if record == nil {
			continue
		}
		friendName := CocoName
		if friendID != CocoID {
			friend := usersByID[friendID]
			if friend == nil {
				continue
			}
			friendName = friend.Name
		}
		record.FriendColors[friendName] = color
	}
	if err := colorRows.Close(); err != nil {
		return err
	}

	clearedRows, err := s.db.Query(`SELECT user_id, conversation_key, cleared_at FROM cleared_at ORDER BY user_id, conversation_key`)
	if err != nil {
		return err
	}
	for clearedRows.Next() {
		var userID, key, value string
		if err := clearedRows.Scan(&userID, &key, &value); err != nil {
			clearedRows.Close()
			return err
		}
		if record := recordsByID[userID]; record != nil {
			record.ClearedAt[key] = value
		}
	}
	if err := clearedRows.Close(); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(s.userBackupPath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return err
	}
	temporary := s.userBackupPath + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, s.userBackupPath)
}

func allUsers(q queryer) ([]*databaseUser, error) {
	rows, err := q.Query(`SELECT id, name, password_hash, signature, show_message_time, parse_latex, theme, created_at
		FROM users ORDER BY created_at, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []*databaseUser{}
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func findUserByName(q queryer, name string) (*databaseUser, error) {
	users, err := allUsers(q)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		if strings.EqualFold(user.Name, strings.TrimSpace(name)) {
			return user, nil
		}
	}
	return nil, nil
}

func userByID(q queryer, id string) (*databaseUser, error) {
	row := q.QueryRow(`SELECT id, name, password_hash, signature, show_message_time, parse_latex, theme, created_at
		FROM users WHERE id = ?`, id)
	user, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return user, err
}

func scanUser(scanner rowScanner) (*databaseUser, error) {
	var user databaseUser
	var showMessageTime, parseLatex int
	var createdAt string
	if err := scanner.Scan(&user.ID, &user.Name, &user.PasswordHash, &user.Signature,
		&showMessageTime, &parseLatex, &user.Settings.Theme, &createdAt); err != nil {
		return nil, err
	}
	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil, err
	}
	user.Settings.ShowMessageTime = showMessageTime != 0
	user.Settings.ParseLatex = parseLatex != 0
	user.CreatedAt = parsedCreatedAt
	return &user, nil
}

func scanMessage(scanner rowScanner) (*Message, error) {
	var message Message
	var sentAt string
	var deliveredAt sql.NullString
	if err := scanner.Scan(&message.ID, &message.Kind, &message.FromID, &message.ToID, &message.GroupID,
		&message.Text, &sentAt, &deliveredAt); err != nil {
		return nil, err
	}
	parsedSentAt, err := time.Parse(time.RFC3339Nano, sentAt)
	if err != nil {
		return nil, err
	}
	message.SentAt = parsedSentAt
	if deliveredAt.Valid {
		parsedDeliveredAt, err := time.Parse(time.RFC3339Nano, deliveredAt.String)
		if err != nil {
			return nil, err
		}
		message.DeliveredAt = &parsedDeliveredAt
	}
	return &message, nil
}

func clearedTimes(q queryer, userID string) (map[string]time.Time, error) {
	rows, err := q.Query(`SELECT conversation_key, cleared_at FROM cleared_at WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]time.Time{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return nil, err
		}
		result[key] = parsed
	}
	return result, rows.Err()
}

func visibleConversation(userID string, message *Message, clearedAt map[string]time.Time, users map[string]*databaseUser, groupAccess map[string]groupAccess) (string, bool) {
	if message.Kind == "group" {
		groupID := message.GroupID
		if groupID == "" {
			groupID = PublicGroupID
		}
		access, member := groupAccess[groupID]
		if !member || !message.SentAt.After(access.HistoryFrom) {
			return "", false
		}
		key := GroupConversationKey(groupID)
		return key, message.SentAt.After(clearedAt[key])
	}
	if message.FromID != userID && message.ToID != userID {
		return "", false
	}
	counterpartID := message.FromID
	if counterpartID == userID {
		counterpartID = message.ToID
	}
	key := "dm:" + counterpartID
	if !message.SentAt.After(clearedAt[key]) {
		return "", false
	}
	name := CocoName
	if counterpartID != CocoID {
		counterpart := users[counterpartID]
		if counterpart == nil {
			return "", false
		}
		name = counterpart.Name
	}
	return "dm:" + name, true
}

func messageView(viewerID string, message *Message, users map[string]*databaseUser) MessageView {
	from := CocoName
	system := false
	if message.FromID == "*" {
		from = "*"
		system = true
	} else if message.FromID != CocoID {
		if user := users[message.FromID]; user != nil {
			from = user.Name
		}
	}
	delivery := "sent"
	if message.FromID == viewerID && message.ToID != "" && message.DeliveredAt == nil {
		delivery = "queued"
	}
	return MessageView{
		ID: message.ID, From: from, Text: message.Text,
		SentAt: message.SentAt.Format(time.RFC3339Nano), Delivery: delivery, System: system,
	}
}

func selfView(user *databaseUser) SelfView {
	return SelfView{ID: user.ID, Name: user.Name, Signature: user.Signature, Settings: user.Settings}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func isUniqueError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}

func randomID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buffer)
}
