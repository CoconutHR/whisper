package chat

import (
	"path/filepath"
	"testing"
)

func TestConversationReadCursorPersistsUnreadCounts(t *testing.T) {
	store := newTestStore(t)
	alice, err := store.Register("readali", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := store.Register("readbob", "password123")
	if err != nil {
		t.Fatal(err)
	}

	first, _, err := store.SendMessage(alice.ID, "private", bob.Name, "first", false)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := store.SendMessage(alice.ID, "private", bob.Name, "second", false)
	if err != nil {
		t.Fatal(err)
	}
	view, err := store.Bootstrap(bob.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if got := view.UnreadCounts["dm:"+alice.Name]; got != 2 {
		t.Fatalf("unread count = %d, want 2", got)
	}

	conversation, err := store.MarkConversationRead(bob.ID, "dm:"+alice.Name, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if conversation != "dm:"+alice.Name {
		t.Fatalf("conversation = %q", conversation)
	}
	if _, err := store.MarkConversationRead(bob.ID, "dm:"+alice.Name, first.ID); err != nil {
		t.Fatal(err)
	}
	view, err = store.Bootstrap(bob.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if got := view.UnreadCounts["dm:"+alice.Name]; got != 0 {
		t.Fatalf("unread count after read = %d, want 0", got)
	}

	if _, err := store.MarkConversationRead(alice.ID, "dm:"+bob.Name, second.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkConversationRead(bob.ID, "dm:coco", second.ID); err == nil {
		t.Fatal("expected a message from another conversation to be rejected")
	}
}

func TestReadCursorBaselinesForPublicAndNewGroupMembers(t *testing.T) {
	store := newTestStore(t)
	alice, err := store.Register("baseali", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.SendGroupMessage(alice.ID, PublicGroupID, "before registration"); err != nil {
		t.Fatal(err)
	}
	bob, err := store.Register("basebob", "password123")
	if err != nil {
		t.Fatal(err)
	}
	view, err := store.Bootstrap(bob.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if got := view.UnreadCounts[GroupConversationKey(PublicGroupID)]; got != 0 {
		t.Fatalf("public history unread count = %d, want 0", got)
	}

	carol, err := store.Register("basecar", "password123")
	if err != nil {
		t.Fatal(err)
	}
	mutation, _, err := store.CreateGroup(alice.ID, "history", "", true, []string{bob.Name})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.SendGroupMessage(alice.ID, mutation.GroupID, "before joining"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.UpdateGroup(alice.ID, mutation.GroupID, "history", "", true,
		[]string{bob.Name, carol.Name}); err != nil {
		t.Fatal(err)
	}
	view, err = store.Bootstrap(carol.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Conversations[GroupConversationKey(mutation.GroupID)]) == 0 {
		t.Fatal("new member should see group history")
	}
	if got := view.UnreadCounts[GroupConversationKey(mutation.GroupID)]; got != 0 {
		t.Fatalf("visible group history unread count = %d, want 0", got)
	}
	if _, _, err := store.SendGroupMessage(alice.ID, mutation.GroupID, "after joining"); err != nil {
		t.Fatal(err)
	}
	view, err = store.Bootstrap(carol.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if got := view.UnreadCounts[GroupConversationKey(mutation.GroupID)]; got != 1 {
		t.Fatalf("new group message unread count = %d, want 1", got)
	}
}

func TestConversationReadV9MigrationSeedsExistingHistory(t *testing.T) {
	directory := t.TempDir()
	config := StoreConfig{
		DatabasePath:   filepath.Join(directory, "whisper.db"),
		UserBackupPath: filepath.Join(directory, "users-backup.json"),
	}
	store, err := NewStore(config)
	if err != nil {
		t.Fatal(err)
	}
	alice, err := store.Register("migali", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := store.Register("migbob", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.SendMessage(alice.ID, "private", bob.Name, "old message", false); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`DELETE FROM conversation_reads`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE meta SET value = '8' WHERE key = 'schema_version'`); err != nil {
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
	view, err := store.Bootstrap(bob.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if got := view.UnreadCounts["dm:"+alice.Name]; got != 0 {
		t.Fatalf("migrated old history unread count = %d, want 0", got)
	}
	var version int
	if err := store.db.QueryRow(`SELECT value FROM meta WHERE key = 'schema_version'`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != schemaVersion {
		t.Fatalf("schema version = %d, want %d", version, schemaVersion)
	}
}
