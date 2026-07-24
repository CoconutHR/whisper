package chat

import (
	"fmt"
	"testing"
)

func TestConversationMessagePagination(t *testing.T) {
	store := newTestStore(t)
	alice, err := store.Register("pageali", "password123")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := store.Register("pagebob", "password123")
	if err != nil {
		t.Fatal(err)
	}

	sent := make([]*Message, 0, 120)
	for index := range 120 {
		message, _, sendErr := store.SendGroupMessage(alice.ID, PublicGroupID, fmt.Sprintf("message-%03d", index))
		if sendErr != nil {
			t.Fatal(sendErr)
		}
		sent = append(sent, message)
	}

	bootstrap, err := store.Bootstrap(bob.ID, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	conversation := GroupConversationKey(PublicGroupID)
	latest := bootstrap.Conversations[conversation]
	if len(latest) != MessagePageSize {
		t.Fatalf("latest page length = %d", len(latest))
	}
	if latest[0].ID != sent[70].ID || latest[49].ID != sent[119].ID {
		t.Fatalf("latest page bounds = %s..%s", latest[0].Text, latest[49].Text)
	}
	if !bootstrap.ConversationHasMore[conversation] {
		t.Fatal("latest page should have older messages")
	}
	if got := bootstrap.UnreadCounts[conversation]; got != 120 {
		t.Fatalf("unread count = %d", got)
	}

	page, err := store.ConversationMessages(bob.ID, conversation, &MessageCursor{
		SentAt: sent[70].SentAt, MessageID: sent[70].ID,
	}, MessagePageSize)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Messages) != MessagePageSize || !page.HasMore {
		t.Fatalf("second page length/hasMore = %d/%v", len(page.Messages), page.HasMore)
	}
	if page.Messages[0].ID != sent[20].ID || page.Messages[49].ID != sent[69].ID {
		t.Fatalf("second page bounds = %s..%s", page.Messages[0].Text, page.Messages[49].Text)
	}

	page, err = store.ConversationMessages(bob.ID, conversation, &MessageCursor{
		SentAt: sent[20].SentAt, MessageID: sent[20].ID,
	}, MessagePageSize)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Messages) != 20 || page.HasMore {
		t.Fatalf("oldest page length/hasMore = %d/%v", len(page.Messages), page.HasMore)
	}
	if page.Messages[0].ID != sent[0].ID || page.Messages[19].ID != sent[19].ID {
		t.Fatalf("oldest page bounds = %s..%s", page.Messages[0].Text, page.Messages[19].Text)
	}
}
