package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"whisper/internal/chat"
)

type recordedPush struct {
	userID  string
	message pushMessage
}

type recordingPushSender struct {
	messages chan recordedPush
}

func (s *recordingPushSender) PublicKey() string {
	return "test-public-key"
}

func (s *recordingPushSender) Send(userID string, message pushMessage) {
	s.messages <- recordedPush{userID: userID, message: message}
}

func TestPushSubscriptionAPIAndMessageDispatch(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{
		"index.html", "styles.css", "app.js", "sw.js",
		"logo-oracle.svg", "logo-oracle-unread.svg",
		"logo-oracle-vector.svg", "logo-oracle-vector-unread.svg",
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := chat.NewStore(chat.StoreConfig{
		DatabasePath: filepath.Join(directory, "whisper.db"), UserBackupPath: filepath.Join(directory, "users.json"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := NewServer(Config{StaticDir: directory}, store, logger)
	recorder := &recordingPushSender{messages: make(chan recordedPush, 1)}
	app.push = recorder
	server := httptest.NewServer(app.Handler())
	defer server.Close()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}

	response := jsonRequest(t, client, http.MethodPost, server.URL+"/api/register", map[string]string{
		"name": "palice", "password": "password123",
	})
	var alice chat.SelfView
	if err := json.NewDecoder(response.Body).Decode(&alice); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	bob, err := store.Register("pbob", "password123")
	if err != nil {
		t.Fatal(err)
	}

	response, err = client.Get(server.URL + "/api/push/config")
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.NewDecoder(response.Body).Decode(&config); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if config["enabled"] != true || config["publicKey"] != "test-public-key" {
		t.Fatalf("push config = %#v", config)
	}

	subscription := chat.PushSubscription{
		Endpoint: "https://push.example.test/subscription",
		Keys:     chat.PushSubscriptionKeys{P256dh: "public-key", Auth: "auth-key"},
	}
	response = jsonRequest(t, client, http.MethodPost, server.URL+"/api/push/subscription", subscription)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("subscribe status = %d", response.StatusCode)
	}
	subscriptions, err := store.PushSubscriptions(alice.ID)
	if err != nil || len(subscriptions) != 1 {
		t.Fatalf("stored subscriptions = %#v, %v", subscriptions, err)
	}

	message, targetID, err := store.SendMessage(alice.ID, "private", "pbob", "hello push", false)
	if err != nil {
		t.Fatal(err)
	}
	app.dispatchMessage(alice.ID, targetID, "pbob", message, "")
	select {
	case pushed := <-recorder.messages:
		if pushed.userID != bob.ID || pushed.message.Conversation != "dm:palice" ||
			pushed.message.MessageID != message.ID || pushed.message.Body != "hello push" {
			t.Fatalf("pushed message = %#v", pushed)
		}
	case <-time.After(time.Second):
		t.Fatal("push message was not dispatched")
	}

	response = jsonRequest(t, client, http.MethodDelete, server.URL+"/api/push/subscription", map[string]string{
		"endpoint": subscription.Endpoint,
	})
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("unsubscribe status = %d", response.StatusCode)
	}
	subscriptions, err = store.PushSubscriptions(alice.ID)
	if err != nil || len(subscriptions) != 0 {
		t.Fatalf("subscriptions after delete = %#v, %v", subscriptions, err)
	}
}

func TestPushSubscriptionRejectsLocalEndpoint(t *testing.T) {
	if err := validatePushSubscription(chat.PushSubscription{
		Endpoint: "https://127.0.0.1/push",
		Keys:     chat.PushSubscriptionKeys{P256dh: "public-key", Auth: "auth-key"},
	}); err == nil {
		t.Fatal("expected a local push endpoint to be rejected")
	}
}

func TestPublicPushIPFilter(t *testing.T) {
	if isPublicPushIP(net.ParseIP("127.0.0.1")) || isPublicPushIP(net.ParseIP("10.0.0.1")) {
		t.Fatal("private push addresses must be rejected")
	}
	if !isPublicPushIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("public push address must be allowed")
	}
}

func TestNormalizeVAPIDSubject(t *testing.T) {
	if got := normalizeVAPIDSubject("mailto:admin@example.com"); got != "admin@example.com" {
		t.Fatalf("normalized mailto subject = %q", got)
	}
	if got := normalizeVAPIDSubject("https://example.com/contact"); got != "https://example.com/contact" {
		t.Fatalf("normalized URL subject = %q", got)
	}
}
