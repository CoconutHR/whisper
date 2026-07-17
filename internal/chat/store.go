package chat

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	CocoID       = "coco"
	CocoName     = "coco"
	MaxNameRunes = 7
	MaxMessage   = 2000
	stateVersion = 2
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

type User struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	PasswordHash string               `json:"passwordHash"`
	Signature    string               `json:"signature"`
	Settings     Settings             `json:"settings"`
	Friends      map[string]bool      `json:"friends"`
	FriendColors map[string]string    `json:"friendColors,omitempty"`
	ClearedAt    map[string]time.Time `json:"clearedAt"`
	CreatedAt    time.Time            `json:"createdAt"`
}

type Message struct {
	ID          string     `json:"id"`
	Kind        string     `json:"kind"`
	FromID      string     `json:"fromId"`
	ToID        string     `json:"toId,omitempty"`
	Text        string     `json:"text"`
	SentAt      time.Time  `json:"sentAt"`
	DeliveredAt *time.Time `json:"deliveredAt,omitempty"`
}

type persistedState struct {
	Version  int              `json:"version"`
	Users    map[string]*User `json:"users"`
	Messages []*Message       `json:"messages"`
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

type Bootstrap struct {
	Self          SelfView                 `json:"self"`
	Members       []MemberView             `json:"members"`
	Friends       []string                 `json:"friends"`
	FriendColors  map[string]string        `json:"friendColors"`
	Conversations map[string][]MessageView `json:"conversations"`
}

type DeliveryNotice struct {
	MessageID string
	SenderID  string
}

type Store struct {
	mu    sync.RWMutex
	path  string
	state persistedState
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path}
	s.state = persistedState{Version: stateVersion, Users: map[string]*User{}, Messages: []*Message{}}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &s.state); err != nil {
		return nil, fmt.Errorf("读取状态文件: %w", err)
	}
	if s.state.Users == nil {
		s.state.Users = map[string]*User{}
	}
	if s.state.Messages == nil {
		s.state.Messages = []*Message{}
	}
	s.state.Version = stateVersion
	for _, user := range s.state.Users {
		initializeUserMaps(user)
		user.Friends[CocoID] = true
	}
	return s, nil
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
	if len(password) < 1 {
		return SelfView{}, errors.New("密码不能为空")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return SelfView{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.userByNameLocked(name) != nil {
		return SelfView{}, ErrNameTaken
	}

	now := time.Now()
	user := &User{
		ID:           randomID(),
		Name:         name,
		PasswordHash: string(hash),
		Settings:     DefaultSettings(),
		Friends:      map[string]bool{CocoID: true},
		FriendColors: map[string]string{},
		ClearedAt:    map[string]time.Time{},
		CreatedAt:    now,
	}
	s.state.Users[user.ID] = user
	delivered := now
	s.state.Messages = append(s.state.Messages, &Message{
		ID: randomID(), Kind: "private", FromID: CocoID, ToID: user.ID,
		Text: "这里的消息仅你自己可见。", SentAt: now, DeliveredAt: &delivered,
	})
	if err := s.saveLocked(); err != nil {
		return SelfView{}, err
	}
	return selfView(user), nil
}

func (s *Store) Authenticate(name, password string) (SelfView, error) {
	s.mu.RLock()
	user := s.userByNameLocked(strings.TrimSpace(name))
	if user == nil {
		s.mu.RUnlock()
		return SelfView{}, ErrUnauthorized
	}
	hash := user.PasswordHash
	view := selfView(user)
	s.mu.RUnlock()

	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return SelfView{}, ErrUnauthorized
	}
	return view, nil
}

func (s *Store) User(id string) (SelfView, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user := s.state.Users[id]
	if user == nil {
		return SelfView{}, ErrNotFound
	}
	return selfView(user), nil
}

func (s *Store) Bootstrap(userID string, online map[string]bool) (Bootstrap, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user := s.state.Users[userID]
	if user == nil {
		return Bootstrap{}, ErrNotFound
	}

	result := Bootstrap{
		Self: selfView(user),
		Members: []MemberView{{
			Name: CocoName, Online: true, Reserved: true, Signature: "仅自己可见",
		}},
		Friends:       []string{},
		FriendColors:  map[string]string{},
		Conversations: map[string][]MessageView{"group": {}, "dm:coco": {}},
	}

	users := make([]*User, 0, len(s.state.Users)-1)
	for _, candidate := range s.state.Users {
		if candidate.ID == userID {
			continue
		}
		users = append(users, candidate)
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Name < users[j].Name })
	for _, candidate := range users {
		result.Members = append(result.Members, MemberView{
			Name: candidate.Name, Online: online[candidate.ID], Signature: candidate.Signature,
		})
		result.Conversations["dm:"+candidate.Name] = []MessageView{}
	}

	for friendID := range user.Friends {
		friendName := ""
		if friendID == CocoID {
			friendName = CocoName
		} else if friend := s.state.Users[friendID]; friend != nil {
			friendName = friend.Name
		}
		if friendName == "" {
			continue
		}
		result.Friends = append(result.Friends, friendName)
		if color := user.FriendColors[friendID]; color != "" {
			result.FriendColors[friendName] = color
		}
	}
	sort.Strings(result.Friends)

	for _, message := range s.state.Messages {
		key, ok := s.visibleConversationLocked(user, message)
		if !ok {
			continue
		}
		result.Conversations[key] = append(
			result.Conversations[key], s.messageViewLocked(userID, message),
		)
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

	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.state.Users[userID]
	if user == nil {
		return SelfView{}, "", ErrNotFound
	}
	if existing := s.userByNameLocked(name); existing != nil && existing.ID != userID {
		return SelfView{}, "", ErrNameTaken
	}
	previousName := user.Name
	user.Name = name
	user.Signature = strings.TrimSpace(signature)
	if err := s.saveLocked(); err != nil {
		return SelfView{}, "", err
	}
	return selfView(user), previousName, nil
}

func (s *Store) UpdatePassword(userID, currentPassword, newPassword string) error {
	if len(newPassword) < 1 {
		return errors.New("新密码不能为空")
	}

	s.mu.RLock()
	user := s.state.Users[userID]
	if user == nil {
		s.mu.RUnlock()
		return ErrNotFound
	}
	hash := user.PasswordHash
	s.mu.RUnlock()
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)) != nil {
		return errors.New("当前密码不正确")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user = s.state.Users[userID]
	if user == nil {
		return ErrNotFound
	}
	user.PasswordHash = string(newHash)
	return s.saveLocked()
}

func (s *Store) UpdateSettings(userID string, settings Settings) error {
	if settings.Theme != "dune" && settings.Theme != "ocean" && settings.Theme != "paper" {
		return errors.New("未知配色方案")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.state.Users[userID]
	if user == nil {
		return ErrNotFound
	}
	user.Settings = settings
	return s.saveLocked()
}

func (s *Store) SendMessage(fromID, scope, targetName, text string, targetOnline bool) (*Message, string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, "", errors.New("消息不能为空")
	}
	if utf8.RuneCountInString(text) > MaxMessage {
		return nil, "", fmt.Errorf("消息不能超过%d个字符", MaxMessage)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	sender := s.state.Users[fromID]
	if sender == nil {
		return nil, "", ErrNotFound
	}

	message := &Message{ID: randomID(), Kind: scope, FromID: fromID, Text: text, SentAt: time.Now()}
	targetID := ""
	if scope == "private" {
		if strings.EqualFold(targetName, CocoName) {
			targetID = CocoID
			delivered := message.SentAt
			message.DeliveredAt = &delivered
		} else {
			target := s.userByNameLocked(targetName)
			if target == nil {
				return nil, "", ErrNotFound
			}
			if target.ID == fromID {
				return nil, "", errors.New("不能向本人发送消息")
			}
			targetID = target.ID
			target.Friends[fromID] = true
			if targetOnline {
				delivered := message.SentAt
				message.DeliveredAt = &delivered
			}
		}
		message.ToID = targetID
		sender.Friends[targetID] = true
	} else if scope != "group" {
		return nil, "", errors.New("未知消息类型")
	}

	s.state.Messages = append(s.state.Messages, message)
	if err := s.saveLocked(); err != nil {
		return nil, "", err
	}
	clone := *message
	return &clone, targetID, nil
}

func (s *Store) ClearConversation(userID, targetName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.state.Users[userID]
	if user == nil {
		return ErrNotFound
	}
	key := "group"
	if targetName != "group" {
		if strings.EqualFold(targetName, CocoName) {
			key = "dm:" + CocoID
		} else {
			target := s.userByNameLocked(targetName)
			if target == nil {
				return ErrNotFound
			}
			key = "dm:" + target.ID
		}
	}
	user.ClearedAt[key] = time.Now()
	return s.saveLocked()
}

func (s *Store) DeleteFriend(userID, name string) error {
	if strings.EqualFold(name, CocoName) {
		return errors.New("coco 不能删除")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.state.Users[userID]
	target := s.userByNameLocked(name)
	if user == nil || target == nil {
		return ErrNotFound
	}
	delete(user.Friends, target.ID)
	delete(user.FriendColors, target.ID)
	return s.saveLocked()
}

func (s *Store) SetFriendColor(userID, name, color string) error {
	color = strings.ToLower(strings.TrimSpace(color))
	if color != "default" && color != "green" && color != "blue" &&
		color != "cyan" && color != "amber" && color != "rose" {
		return errors.New("未知的好友消息颜色")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.state.Users[userID]
	if user == nil {
		return ErrNotFound
	}
	targetID := CocoID
	if !strings.EqualFold(name, CocoName) {
		target := s.userByNameLocked(name)
		if target == nil {
			return ErrNotFound
		}
		targetID = target.ID
	}
	if !user.Friends[targetID] {
		return errors.New("只能设置好友的消息颜色")
	}
	if color == "default" {
		delete(user.FriendColors, targetID)
	} else {
		user.FriendColors[targetID] = color
	}
	return s.saveLocked()
}

func (s *Store) MarkDelivered(userID string) ([]DeliveryNotice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Users[userID] == nil {
		return nil, ErrNotFound
	}
	now := time.Now()
	notices := []DeliveryNotice{}
	for _, message := range s.state.Messages {
		if message.ToID == userID && message.DeliveredAt == nil {
			message.DeliveredAt = &now
			notices = append(notices, DeliveryNotice{MessageID: message.ID, SenderID: message.FromID})
		}
	}
	if len(notices) > 0 {
		if err := s.saveLocked(); err != nil {
			return nil, err
		}
	}
	return notices, nil
}

func (s *Store) MessageView(viewerID string, message *Message) MessageView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.messageViewLocked(viewerID, message)
}

func (s *Store) NameForID(id string) string {
	if id == CocoID {
		return CocoName
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if user := s.state.Users[id]; user != nil {
		return user.Name
	}
	return ""
}

func (s *Store) IDForName(name string) string {
	if strings.EqualFold(name, CocoName) {
		return CocoID
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if user := s.userByNameLocked(name); user != nil {
		return user.ID
	}
	return ""
}

func (s *Store) visibleConversationLocked(user *User, message *Message) (string, bool) {
	if message.Kind == "group" {
		return "group", message.SentAt.After(user.ClearedAt["group"])
	}
	if message.FromID != user.ID && message.ToID != user.ID {
		return "", false
	}
	counterpartID := message.FromID
	if counterpartID == user.ID {
		counterpartID = message.ToID
	}
	key := "dm:" + counterpartID
	if !message.SentAt.After(user.ClearedAt[key]) {
		return "", false
	}
	name := CocoName
	if counterpartID != CocoID {
		counterpart := s.state.Users[counterpartID]
		if counterpart == nil {
			return "", false
		}
		name = counterpart.Name
	}
	return "dm:" + name, true
}

func (s *Store) messageViewLocked(viewerID string, message *Message) MessageView {
	from := CocoName
	if message.FromID != CocoID {
		if user := s.state.Users[message.FromID]; user != nil {
			from = user.Name
		}
	}
	delivery := "sent"
	if message.FromID == viewerID && message.ToID != "" && message.DeliveredAt == nil {
		delivery = "queued"
	}
	return MessageView{
		ID: message.ID, From: from, Text: message.Text,
		SentAt: message.SentAt.Format(time.RFC3339Nano), Delivery: delivery,
	}
}

func (s *Store) userByNameLocked(name string) *User {
	for _, user := range s.state.Users {
		if strings.EqualFold(user.Name, name) {
			return user
		}
	}
	return nil
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	temporary := s.path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, s.path)
}

func initializeUserMaps(user *User) {
	if user.Friends == nil {
		user.Friends = map[string]bool{}
	}
	if user.ClearedAt == nil {
		user.ClearedAt = map[string]time.Time{}
	}
	if user.FriendColors == nil {
		user.FriendColors = map[string]string{}
	}
	if user.Settings.Theme == "" {
		user.Settings = DefaultSettings()
	}
}

func selfView(user *User) SelfView {
	return SelfView{ID: user.ID, Name: user.Name, Signature: user.Signature, Settings: user.Settings}
}

func randomID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buffer)
}
