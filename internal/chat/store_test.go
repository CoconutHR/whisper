package chat

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
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
	bobView, err := store.Bootstrap(bob.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range bobView.Conversations["dm:coco"] {
		if item.Text == "仅自己可见" {
			t.Fatal("bob can see alice's coco message")
		}
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
