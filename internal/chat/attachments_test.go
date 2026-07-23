package chat

import (
	"errors"
	"testing"
	"time"
)

func readyAttachment(t *testing.T, store *Store, userID, name, contentType string, size int64) Attachment {
	t.Helper()
	attachment, err := store.CreateAttachmentDraft(userID, name, contentType, size)
	if err != nil {
		t.Fatalf("CreateAttachmentDraft: %v", err)
	}
	attachment, err = store.CompleteAttachmentDraft(userID, attachment.ID, size, contentType)
	if err != nil {
		t.Fatalf("CompleteAttachmentDraft: %v", err)
	}
	return attachment
}

func TestAttachmentViewMediaClassification(t *testing.T) {
	cases := []struct {
		contentType string
		kind        string
		previewable bool
		streamable  bool
	}{
		{"image/png", "image", true, false},
		{"video/mp4", "video", true, true},
		{"video/quicktime", "video", true, true},
		{"audio/mpeg", "audio", true, true},
		{"audio/x-m4a", "audio", true, true},
		{"audio/x-aac", "audio", true, true},
		{"audio/flac", "audio", true, true},
		{"audio/x-flac", "audio", true, true},
		{"application/pdf", "document", true, false},
		{"application/x-pdf", "document", true, false},
		{"text/plain; charset=utf-8", "document", true, false},
		{"application/msword", "document", true, false},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", "document", true, false},
		{"application/vnd.ms-excel", "document", true, false},
		{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "document", true, false},
		{"text/html", "file", false, false},
		{"image/svg+xml", "file", false, false},
	}
	for _, testCase := range cases {
		view := attachmentView(Attachment{ID: "asset", Name: "asset", ContentType: testCase.contentType, Size: 1024})
		if view.Kind != testCase.kind || view.Inline != (testCase.kind == "image") {
			t.Errorf("attachmentView(%q) = %#v", testCase.contentType, view)
		}
		if got := IsBrowserPreviewableContentType(testCase.contentType); got != testCase.previewable {
			t.Errorf("IsBrowserPreviewableContentType(%q) = %v", testCase.contentType, got)
		}
		if got := IsStreamableMediaContentType(testCase.contentType); got != testCase.streamable {
			t.Errorf("IsStreamableMediaContentType(%q) = %v", testCase.contentType, got)
		}
	}
	largeImage := attachmentView(Attachment{ID: "large", Name: "large.png", ContentType: "image/png", Size: InlineImageMaxSize + 1})
	if largeImage.Inline {
		t.Fatalf("large image view = %#v", largeImage)
	}
}

func TestAttachmentMessageLifecycleAndPrivateAccess(t *testing.T) {
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

	attachment := readyAttachment(t, store, alice.ID, "../报告.pdf", "application/pdf", 4096)
	if attachment.Name != "报告.pdf" {
		t.Fatalf("normalized name = %q", attachment.Name)
	}
	message, _, err := store.SendMessageContent(alice.ID, "private", "bob", MessageContent{
		AttachmentIDs: []string{attachment.ID},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(message.Attachments) != 1 || message.Attachments[0].URL != "/api/attachments/"+attachment.ID {
		t.Fatalf("message attachments = %#v", message.Attachments)
	}
	if _, err := store.OwnedAttachmentDraft(alice.ID, attachment.ID); !errors.Is(err, ErrAttachmentForbidden) {
		t.Fatalf("attached draft remained mutable: %v", err)
	}
	if _, err := store.AttachmentForViewer(alice.ID, attachment.ID); err != nil {
		t.Fatalf("sender cannot view attachment: %v", err)
	}
	if _, err := store.AttachmentForViewer(bob.ID, attachment.ID); err != nil {
		t.Fatalf("recipient cannot view attachment: %v", err)
	}
	if _, err := store.AttachmentForViewer(carol.ID, attachment.ID); !errors.Is(err, ErrAttachmentForbidden) {
		t.Fatalf("unrelated user access error = %v", err)
	}

	view, err := store.Bootstrap(bob.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	messages := view.Conversations["dm:alice"]
	if len(messages) != 1 || len(messages[0].Attachments) != 1 {
		t.Fatalf("bootstrap messages = %#v", messages)
	}
	if err := store.ClearConversation(bob.ID, "alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AttachmentForViewer(bob.ID, attachment.ID); !errors.Is(err, ErrAttachmentForbidden) {
		t.Fatalf("cleared recipient still has access: %v", err)
	}
}

func TestAttachmentBindingRollsBackMessageOnPermissionFailure(t *testing.T) {
	store := newTestStore(t)
	alice, err := store.Register("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := store.Register("bob", "password123")
	if err != nil {
		t.Fatal(err)
	}
	attachment := readyAttachment(t, store, alice.ID, "secret.txt", "text/plain", 12)

	_, _, err = store.SendMessageContent(bob.ID, "private", "alice", MessageContent{
		Text: "不应写入", AttachmentIDs: []string{attachment.ID},
	}, true)
	if !errors.Is(err, ErrAttachmentForbidden) {
		t.Fatalf("cross-user attachment error = %v", err)
	}
	view, err := store.Bootstrap(alice.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	for _, message := range view.Conversations["dm:bob"] {
		if message.Text == "不应写入" {
			t.Fatal("message insert was not rolled back")
		}
	}
	if draft, err := store.OwnedAttachmentDraft(alice.ID, attachment.ID); err != nil || draft.Status != "ready" {
		t.Fatalf("attachment changed after rollback: %#v, %v", draft, err)
	}
}

func TestStickerValidation(t *testing.T) {
	store := newTestStore(t)
	alice, err := store.Register("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.SendGroupMessageContent(alice.ID, PublicGroupID, MessageContent{
		Sticker: "great-job",
	}); err != nil {
		t.Fatalf("valid sticker: %v", err)
	}
	if _, _, err := store.SendGroupMessageContent(alice.ID, PublicGroupID, MessageContent{
		Sticker: "unknown-sticker",
	}); err == nil {
		t.Fatal("unknown sticker was accepted")
	}
	if _, _, err := store.SendGroupMessageContent(alice.ID, PublicGroupID, MessageContent{
		Text: "text", Sticker: "great-job",
	}); err == nil {
		t.Fatal("combined text and sticker was accepted")
	}
}

func TestDissolvedGroupAttachmentBecomesCleanupCandidate(t *testing.T) {
	store := newTestStore(t)
	alice, err := store.Register("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Register("bob", "password123"); err != nil {
		t.Fatal(err)
	}
	mutation, _, err := store.CreateGroup(alice.ID, "files", "", true, []string{"bob"})
	if err != nil {
		t.Fatal(err)
	}
	attachment := readyAttachment(t, store, alice.ID, "group.txt", "text/plain", 16)
	if _, _, err := store.SendGroupMessageContent(alice.ID, mutation.GroupID, MessageContent{
		AttachmentIDs: []string{attachment.ID},
	}); err != nil {
		t.Fatal(err)
	}
	if drafts, err := store.ExpiredAttachmentDrafts(time.Now().Add(-24 * time.Hour)); err != nil || len(drafts) != 0 {
		t.Fatalf("referenced attachment selected for cleanup: %#v, %v", drafts, err)
	}
	if _, err := store.DissolveGroup(alice.ID, mutation.GroupID); err != nil {
		t.Fatal(err)
	}
	drafts, err := store.ExpiredAttachmentDrafts(time.Now().Add(-24 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(drafts) != 1 || drafts[0].ID != attachment.ID {
		t.Fatalf("orphan cleanup candidates = %#v", drafts)
	}
	if err := store.DeleteExpiredAttachmentDraft(attachment.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AttachmentForViewer(alice.ID, attachment.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("orphan metadata was not deleted: %v", err)
	}
}

func TestCustomStickerCollectionAndSending(t *testing.T) {
	store := newTestStore(t)
	alice, err := store.Register("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := store.Register("bob", "password123")
	if err != nil {
		t.Fatal(err)
	}
	stickerAsset := readyAttachment(t, store, alice.ID, "hello.webp", "image/webp", 2048)
	sticker, err := store.AddStickerDraft(alice.ID, stickerAsset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sticker.Kind != "image" || !sticker.Inline {
		t.Fatalf("sticker view = %#v", sticker)
	}
	message, _, err := store.SendMessageContent(alice.ID, "private", "bob", MessageContent{
		StickerAttachmentID: stickerAsset.ID,
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if message.StickerAttachment == nil || message.StickerAttachment.ID != stickerAsset.ID {
		t.Fatalf("message sticker = %#v", message.StickerAttachment)
	}
	if _, err := store.FavoriteSticker(bob.ID, stickerAsset.ID); err != nil {
		t.Fatalf("favorite received sticker: %v", err)
	}
	bobView, err := store.Bootstrap(bob.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bobView.Stickers) != 1 || bobView.Stickers[0].ID != stickerAsset.ID {
		t.Fatalf("bob stickers = %#v", bobView.Stickers)
	}
	if got := bobView.Conversations["dm:alice"]; len(got) != 1 || got[0].StickerAttachment == nil {
		t.Fatalf("bob sticker message = %#v", got)
	}
	if err := store.RemoveSticker(alice.ID, stickerAsset.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AttachmentForViewer(bob.ID, stickerAsset.ID); err != nil {
		t.Fatalf("favorite lost access after sender removed collection: %v", err)
	}
}

func TestForwardAndRecallAttachment(t *testing.T) {
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
	attachment := readyAttachment(t, store, alice.ID, "photo.png", "image/png", 4096)
	original, _, err := store.SendMessageContent(alice.ID, "private", "bob", MessageContent{
		AttachmentIDs: []string{attachment.ID},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	forwarded, _, err := store.SendMessageContent(bob.ID, "private", "carol", MessageContent{
		ForwardAttachmentID: attachment.ID,
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(forwarded.Attachments) != 1 || forwarded.Attachments[0].ID != attachment.ID {
		t.Fatalf("forwarded attachments = %#v", forwarded.Attachments)
	}
	if _, err := store.AttachmentForViewer(carol.ID, attachment.ID); err != nil {
		t.Fatalf("forward recipient cannot access attachment: %v", err)
	}
	result, err := store.RecallMessage(alice.ID, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Message.RecalledAt == nil {
		t.Fatal("recall timestamp missing")
	}
	bobView, err := store.Bootstrap(bob.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	messages := bobView.Conversations["dm:alice"]
	if len(messages) != 1 || !messages[0].Recalled || len(messages[0].Attachments) != 0 {
		t.Fatalf("recalled message view = %#v", messages)
	}
	if _, err := store.RecallMessage(bob.ID, original.ID); !errors.Is(err, ErrMessageForbidden) {
		t.Fatalf("non-sender recall error = %v", err)
	}
}

func TestRecalledAttachmentWithoutOtherReferencesIsNotVisible(t *testing.T) {
	store := newTestStore(t)
	alice, err := store.Register("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := store.Register("bob", "password123")
	if err != nil {
		t.Fatal(err)
	}
	attachment := readyAttachment(t, store, alice.ID, "private.png", "image/png", 1024)
	message, _, err := store.SendMessageContent(alice.ID, "private", "bob", MessageContent{
		AttachmentIDs: []string{attachment.ID},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecallMessage(alice.ID, message.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AttachmentForViewer(bob.ID, attachment.ID); !errors.Is(err, ErrAttachmentForbidden) {
		t.Fatalf("recalled attachment access error = %v", err)
	}
	candidates, err := store.ExpiredAttachmentDrafts(time.Now().Add(-24 * time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].ID != attachment.ID {
		t.Fatalf("recalled orphan candidates = %#v", candidates)
	}
}

func TestRecallWindowExpires(t *testing.T) {
	store := newTestStore(t)
	alice, err := store.Register("alice", "password123")
	if err != nil {
		t.Fatal(err)
	}
	message, _, err := store.SendGroupMessage(alice.ID, PublicGroupID, "too old")
	if err != nil {
		t.Fatal(err)
	}
	oldSentAt := time.Now().Add(-RecallWindow - time.Second).Format(time.RFC3339Nano)
	if _, err := store.db.Exec(`UPDATE messages SET sent_at = ? WHERE id = ?`, oldSentAt, message.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecallMessage(alice.ID, message.ID); !errors.Is(err, ErrRecallExpired) {
		t.Fatalf("expired recall error = %v", err)
	}
}
