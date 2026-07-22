package chat

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMessageAttachmentV7MigrationRemovesAssetUniqueness(t *testing.T) {
	directory := t.TempDir()
	databasePath := filepath.Join(directory, "whisper.db")
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE message_attachments (
		message_id TEXT NOT NULL,
		attachment_id TEXT NOT NULL UNIQUE,
		position INTEGER NOT NULL,
		PRIMARY KEY (message_id, attachment_id)
	)`); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := NewStore(StoreConfig{
		DatabasePath: databasePath, UserBackupPath: filepath.Join(directory, "users-backup.json"),
	})
	if err != nil {
		t.Fatalf("migrate v6 database: %v", err)
	}
	defer store.Close()
	var tableSQL string
	if err := store.db.QueryRow(`SELECT sql FROM sqlite_master
		WHERE type = 'table' AND name = 'message_attachments'`).Scan(&tableSQL); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToUpper(tableSQL), "ATTACHMENT_ID TEXT NOT NULL UNIQUE") {
		t.Fatalf("attachment uniqueness remains after migration: %s", tableSQL)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	directory := t.TempDir()
	store, err := NewStore(StoreConfig{
		DatabasePath:   filepath.Join(directory, "whisper.db"),
		UserBackupPath: filepath.Join(directory, "users-backup.json"),
	})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestRegisterRejectsReservedName(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Register("coco", "password123"); err == nil {
		t.Fatal("expected coco to be rejected")
	}
	if _, err := store.Register("一二三四五六七八", "password123"); err == nil {
		t.Fatal("expected an eight-character name to be rejected")
	}
}

func TestOneCharacterPassword(t *testing.T) {
	store := newTestStore(t)
	user, err := store.Register("short", "x")
	if err != nil {
		t.Fatalf("register with one-character password: %v", err)
	}
	if _, err := store.Authenticate("short", "x"); err != nil {
		t.Fatalf("authenticate with one-character password: %v", err)
	}
	if err := store.UpdatePassword(user.ID, "x", "y"); err != nil {
		t.Fatalf("update to one-character password: %v", err)
	}
	if _, err := store.Authenticate("short", "y"); err != nil {
		t.Fatalf("authenticate with updated password: %v", err)
	}
	if _, err := store.Register("empty", ""); err == nil {
		t.Fatal("expected an empty password to be rejected")
	}
}

func TestFullMessageTimeSetting(t *testing.T) {
	store := newTestStore(t)
	user, err := store.Register("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if user.Settings.FullMessageTime {
		t.Fatal("full message time should be disabled by default")
	}

	settings := user.Settings
	settings.FullMessageTime = true
	if err := store.UpdateSettings(user.ID, settings); err != nil {
		t.Fatal(err)
	}
	view, err := store.Bootstrap(user.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if !view.Self.Settings.FullMessageTime {
		t.Fatal("full message time setting was not persisted")
	}
}

func TestPrivateDeliveryAndCocoIsolation(t *testing.T) {
	store := newTestStore(t)
	alice, err := store.Register("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := store.Register("bob", "password123")
	if err != nil {
		t.Fatal(err)
	}

	message, targetID, err := store.SendMessage(alice.ID, "private", "bob", "离线消息", false)
	if err != nil {
		t.Fatal(err)
	}
	if targetID != bob.ID || message.DeliveredAt != nil {
		t.Fatalf("unexpected queued message: target=%q delivered=%v", targetID, message.DeliveredAt)
	}
	bobView, err := store.Bootstrap(bob.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(bobView.Friends, "alice") {
		t.Fatalf("sender was not added to receiver friends: %#v", bobView.Friends)
	}

	aliceView, err := store.Bootstrap(alice.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if got := aliceView.Conversations["dm:bob"][0].Delivery; got != "queued" {
		t.Fatalf("delivery = %q, want queued", got)
	}

	notices, err := store.MarkDelivered(bob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(notices) != 1 || notices[0].SenderID != alice.ID {
		t.Fatalf("unexpected delivery notices: %#v", notices)
	}

	if _, _, err := store.SendMessage(alice.ID, "private", CocoName, "仅自己可见", true); err != nil {
		t.Fatal(err)
	}
	bobView, err = store.Bootstrap(bob.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range bobView.Conversations["dm:coco"] {
		if item.Text == "仅自己可见" {
			t.Fatal("bob can see alice's coco message")
		}
	}
}

func TestGroupLifecycleAndHistoryVisibility(t *testing.T) {
	store := newTestStore(t)
	alice, err := store.Register("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := store.Register("bob", "password123")
	if err != nil {
		t.Fatal(err)
	}
	carol, err := store.Register("carol", "password123")
	if err != nil {
		t.Fatal(err)
	}

	mutation, group, err := store.CreateGroup(alice.ID, "项目组", "一起推进", false, []string{"bob"})
	if err != nil {
		t.Fatal(err)
	}
	if !group.IsOwner || group.System || mutation.GroupID == PublicGroupID {
		t.Fatalf("unexpected created group: %#v", group)
	}
	if _, _, err := store.CreateGroup(alice.ID, PublicGroupName, "", false, []string{"bob"}); !errors.Is(err, ErrGroupNameReserved) {
		t.Fatalf("reserved group name error = %v", err)
	}

	oldMessage, _, err := store.SendGroupMessage(alice.ID, mutation.GroupID, "加入前的消息")
	if err != nil {
		t.Fatal(err)
	}
	if oldMessage.GroupID != mutation.GroupID {
		t.Fatalf("message group id = %q", oldMessage.GroupID)
	}
	_, _, err = store.UpdateGroup(alice.ID, mutation.GroupID, "项目组", "一起推进", false, []string{"bob", "carol"})
	if err != nil {
		t.Fatal(err)
	}
	carolView, err := store.Bootstrap(carol.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(carolView.Conversations[GroupConversationKey(mutation.GroupID)]); got != 0 {
		t.Fatalf("history should be hidden for new member, got %d messages", got)
	}
	if _, _, err := store.UpdateGroup(alice.ID, mutation.GroupID, "项目组", "一起推进", false, []string{"bob"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.UpdateGroup(alice.ID, mutation.GroupID, "项目组", "一起推进", true, []string{"bob", "carol"}); err != nil {
		t.Fatal(err)
	}
	carolView, err = store.Bootstrap(carol.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(carolView.Conversations[GroupConversationKey(mutation.GroupID)]); got == 0 {
		t.Fatal("history-visible setting did not grant history to a new member")
	}
	if _, _, err := store.SendGroupMessage(alice.ID, mutation.GroupID, "加入后的消息"); err != nil {
		t.Fatal(err)
	}
	carolView, err = store.Bootstrap(carol.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(carolView.Conversations[GroupConversationKey(mutation.GroupID)]); got == 0 {
		t.Fatal("new member cannot see messages sent after joining")
	}
	if _, _, err := store.UpdateGroup(bob.ID, mutation.GroupID, "越权修改", "", false, []string{"alice"}); !errors.Is(err, ErrGroupForbidden) {
		t.Fatalf("non-owner update error = %v", err)
	}
	if _, err := store.LeaveGroup(alice.ID, mutation.GroupID); err == nil {
		t.Fatal("group owner should not leave without transferring ownership")
	}
	if _, err := store.DissolveGroup(alice.ID, mutation.GroupID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Bootstrap(carol.ID, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
}

func TestRenameAndPasswordUpdate(t *testing.T) {
	store := newTestStore(t)
	alice, err := store.Register("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := store.Register("bob", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.SendMessage(alice.ID, "private", "bob", "hello", true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.UpdateProfile(alice.ID, "alice2", "签名 😊"); err != nil {
		t.Fatal(err)
	}

	bobView, err := store.Bootstrap(bob.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bobView.Conversations["dm:alice2"]) != 1 {
		t.Fatalf("renamed conversation missing: %#v", bobView.Conversations)
	}
	if err := store.UpdatePassword(alice.ID, "password123", "new-password-456"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Authenticate("alice2", "new-password-456"); err != nil {
		t.Fatalf("authenticate after password update: %v", err)
	}
}

func TestFriendColorPersistsAcrossRename(t *testing.T) {
	store := newTestStore(t)
	alice, err := store.Register("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := store.Register("bob", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.SendMessage(alice.ID, "private", "bob", "hello", true); err != nil {
		t.Fatal(err)
	}
	if err := store.SetFriendColor(bob.ID, "alice", "blue"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.UpdateProfile(alice.ID, "alice2", ""); err != nil {
		t.Fatal(err)
	}

	view, err := store.Bootstrap(bob.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if got := view.FriendColors["alice2"]; got != "blue" {
		t.Fatalf("friend color = %q, want blue", got)
	}
	if err := store.SetFriendColor(bob.ID, "alice2", "neon"); err == nil {
		t.Fatal("expected an unknown color to be rejected")
	}
	if err := store.DeleteFriend(bob.ID, "alice2"); err != nil {
		t.Fatal(err)
	}
	view, err = store.Bootstrap(bob.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := view.FriendColors["alice2"]; exists {
		t.Fatal("friend color remained after deleting the friend")
	}
}

func TestSQLitePersistsAcrossRestart(t *testing.T) {
	directory := t.TempDir()
	config := StoreConfig{
		DatabasePath:   filepath.Join(directory, "whisper.db"),
		UserBackupPath: filepath.Join(directory, "users-backup.json"),
	}
	store, err := NewStore(config)
	if err != nil {
		t.Fatal(err)
	}
	alice, err := store.Register("alice", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.SendMessage(alice.ID, "group", "", "persisted", true); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	store, err = NewStore(config)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.Authenticate("alice", "secret"); err != nil {
		t.Fatal(err)
	}
	view, err := store.Bootstrap(alice.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Conversations[GroupConversationKey(PublicGroupID)]) != 1 || view.Conversations[GroupConversationKey(PublicGroupID)][0].Text != "persisted" {
		t.Fatalf("messages after restart = %#v", view.Conversations[GroupConversationKey(PublicGroupID)])
	}
}

func TestSessionsPersistAcrossRestart(t *testing.T) {
	directory := t.TempDir()
	config := StoreConfig{
		DatabasePath:   filepath.Join(directory, "whisper.db"),
		UserBackupPath: filepath.Join(directory, "users-backup.json"),
	}
	store, err := NewStore(config)
	if err != nil {
		t.Fatal(err)
	}
	user, err := store.Register("alice", "secret")
	if err != nil {
		t.Fatal(err)
	}
	expiresAt := time.Now().Add(time.Hour)
	if err := store.CreateSession("session-token", user.ID, expiresAt); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	store, err = NewStore(config)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if got, ok, err := store.SessionUser("session-token", time.Now()); err != nil || !ok || got != user.ID {
		t.Fatalf("persisted session = %q, %v, %v", got, ok, err)
	}
	if err := store.CreateSession("expired-token", user.ID, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.SessionUser("expired-token", time.Now()); err != nil || ok {
		t.Fatalf("expired session = %v, %v", ok, err)
	}
	if err := store.DeleteSession("session-token"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.SessionUser("session-token", time.Now()); err != nil || ok {
		t.Fatalf("deleted session = %v, %v", ok, err)
	}
}

func TestPlaintextUserBackup(t *testing.T) {
	directory := t.TempDir()
	backupPath := filepath.Join(directory, "users-backup.json")
	store, err := NewStore(StoreConfig{
		DatabasePath: filepath.Join(directory, "whisper.db"), UserBackupPath: backupPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	user, err := store.Register("alice", "plain-text-password")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.UpdateProfile(user.ID, "alice2", "signature"); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("backup permissions = %o, want 600", got)
	}
	data, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	var backup plaintextBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		t.Fatal(err)
	}
	if len(backup.Users) != 1 {
		t.Fatalf("backup users = %d, want 1", len(backup.Users))
	}
	got := backup.Users[0]
	if got.Name != "alice2" || got.Password != "plain-text-password" || !got.PasswordKnown {
		t.Fatalf("unexpected backup user: %#v", got)
	}
	if got.PasswordHash == "" || got.Signature != "signature" {
		t.Fatalf("incomplete backup user: %#v", got)
	}
}

func TestLegacyJSONIsIgnored(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "state.json"), []byte("not valid json"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(StoreConfig{
		DatabasePath:   filepath.Join(directory, "whisper.db"),
		UserBackupPath: filepath.Join(directory, "users-backup.json"),
	})
	if err != nil {
		t.Fatalf("legacy JSON should be ignored: %v", err)
	}
	defer store.Close()
	if _, err := store.Register("fresh", "x"); err != nil {
		t.Fatal(err)
	}
}
