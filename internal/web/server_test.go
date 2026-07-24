package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"whisper/internal/blob"
	"whisper/internal/chat"
)

func newTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	return newTestServerWithObjects(t, nil)
}

func newTestServerWithObjects(t *testing.T, objects blob.Store) (*httptest.Server, *http.Client) {
	t.Helper()
	directory := t.TempDir()
	for _, name := range []string{"index.html", "styles.css", "app.js"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := chat.NewStore(chat.StoreConfig{
		DatabasePath:   filepath.Join(directory, "whisper.db"),
		UserBackupPath: filepath.Join(directory, "users-backup.json"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(NewServer(Config{StaticDir: directory, ObjectStore: objects}, store, logger).Handler())
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return server, &http.Client{Jar: jar}
}

type fakeObjectStore struct {
	metadata map[string]blob.ObjectMetadata
	deleted  []string
	lastKey  string
}

func newFakeObjectStore() *fakeObjectStore {
	return &fakeObjectStore{metadata: map[string]blob.ObjectMetadata{}}
}

func (s *fakeObjectStore) UploadOrigin() string {
	return "https://example-account.r2.cloudflarestorage.com"
}

func (s *fakeObjectStore) PresignPut(_ context.Context, key, contentType string, size int64, ttl time.Duration) (blob.PresignedRequest, error) {
	s.lastKey = key
	return blob.PresignedRequest{
		URL: "https://upload.example.test/object", Headers: map[string]string{"Content-Type": contentType},
		ExpiresAt: time.Now().Add(ttl),
	}, nil
}

func (s *fakeObjectStore) PresignGet(_ context.Context, key, _ string, ttl time.Duration) (blob.PresignedRequest, error) {
	return blob.PresignedRequest{URL: "https://download.example.test/" + key, ExpiresAt: time.Now().Add(ttl)}, nil
}

func (s *fakeObjectStore) Head(_ context.Context, key string) (blob.ObjectMetadata, error) {
	metadata, ok := s.metadata[key]
	if !ok {
		return blob.ObjectMetadata{}, errors.New("object not found")
	}
	return metadata, nil
}

func (s *fakeObjectStore) Delete(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	delete(s.metadata, key)
	return nil
}

func jsonRequest(t *testing.T, client *http.Client, method, address string, body any) *http.Response {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(method, address, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func TestAuthenticationAndWebSocketMessage(t *testing.T) {
	server, client := newTestServer(t)
	defer server.Close()

	response := jsonRequest(t, client, http.MethodPost, server.URL+"/api/register", map[string]string{
		"name": "alice", "password": "password123",
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", response.StatusCode)
	}
	_ = response.Body.Close()

	response, err := client.Get(server.URL + "/api/bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("bootstrap status = %d", response.StatusCode)
	}
	_ = response.Body.Close()

	parsed, _ := url.Parse(server.URL)
	headers := http.Header{}
	for _, cookie := range client.Jar.Cookies(parsed) {
		headers.Add("Cookie", cookie.String())
	}
	websocketURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	connection, _, err := websocket.DefaultDialer.Dial(websocketURL, headers)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()

	if err := connection.WriteJSON(map[string]string{
		"type": "message", "requestId": "request-1", "scope": "group", "text": "hello",
	}); err != nil {
		t.Fatal(err)
	}
	for range 4 {
		var event map[string]any
		if err := connection.ReadJSON(&event); err != nil {
			t.Fatal(err)
		}
		if event["type"] == "message" {
			if event["conversation"] != chat.GroupConversationKey(chat.PublicGroupID) {
				t.Fatalf("conversation = %#v", event["conversation"])
			}
			if event["requestId"] != "request-1" {
				t.Fatalf("request id = %#v", event["requestId"])
			}
			return
		}
	}
	t.Fatal("group message event not received")
}

func TestConversationReadAPI(t *testing.T) {
	server, client := newTestServer(t)
	defer server.Close()
	response := jsonRequest(t, client, http.MethodPost, server.URL+"/api/register", map[string]string{
		"name": "reader", "password": "password123",
	})
	_ = response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", response.StatusCode)
	}

	response, err := client.Get(server.URL + "/api/bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	var bootstrap chat.Bootstrap
	if err := json.NewDecoder(response.Body).Decode(&bootstrap); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	cocoMessages := bootstrap.Conversations["dm:coco"]
	if len(cocoMessages) == 0 {
		t.Fatal("missing welcome message")
	}
	response = jsonRequest(t, client, http.MethodPost, server.URL+"/api/conversations/read", map[string]string{
		"conversation": "dm:coco", "messageId": cocoMessages[len(cocoMessages)-1].ID,
	})
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("read status = %d", response.StatusCode)
	}
}

func TestWebSocketMessageRecall(t *testing.T) {
	server, client := newTestServer(t)
	defer server.Close()
	response := jsonRequest(t, client, http.MethodPost, server.URL+"/api/register", map[string]string{
		"name": "alice", "password": "password123",
	})
	_ = response.Body.Close()
	parsed, _ := url.Parse(server.URL)
	headers := http.Header{}
	for _, cookie := range client.Jar.Cookies(parsed) {
		headers.Add("Cookie", cookie.String())
	}
	connection, _, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(server.URL, "http")+"/ws", headers,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	_ = connection.SetReadDeadline(time.Now().Add(3 * time.Second))
	if err := connection.WriteJSON(map[string]string{
		"type": "message", "requestId": "send-1", "scope": "group", "text": "recall me",
	}); err != nil {
		t.Fatal(err)
	}
	messageID := ""
	for messageID == "" {
		var event struct {
			Type    string `json:"type"`
			Message struct {
				ID string `json:"id"`
			} `json:"message"`
		}
		if err := connection.ReadJSON(&event); err != nil {
			t.Fatal(err)
		}
		if event.Type == "message" {
			messageID = event.Message.ID
		}
	}
	if err := connection.WriteJSON(map[string]string{
		"type": "recall", "requestId": "recall-1", "messageId": messageID,
	}); err != nil {
		t.Fatal(err)
	}
	for {
		var event map[string]any
		if err := connection.ReadJSON(&event); err != nil {
			t.Fatal(err)
		}
		if event["type"] != "message_recalled" {
			continue
		}
		if event["requestId"] != "recall-1" || event["messageId"] != messageID {
			t.Fatalf("recall event = %#v", event)
		}
		break
	}

	response, err = client.Get(server.URL + "/api/bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var bootstrap chat.Bootstrap
	if err := json.NewDecoder(response.Body).Decode(&bootstrap); err != nil {
		t.Fatal(err)
	}
	messages := bootstrap.Conversations[chat.GroupConversationKey(chat.PublicGroupID)]
	if len(messages) != 1 || messages[0].ID != messageID || !messages[0].Recalled || messages[0].Text != "" {
		t.Fatalf("recalled bootstrap messages = %#v", messages)
	}
}

func TestFriendColorAPI(t *testing.T) {
	server, client := newTestServer(t)
	defer server.Close()

	response := jsonRequest(t, client, http.MethodPost, server.URL+"/api/register", map[string]string{
		"name": "alice", "password": "password123",
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", response.StatusCode)
	}
	_ = response.Body.Close()

	response = jsonRequest(t, client, http.MethodPatch, server.URL+"/api/friends/color", map[string]string{
		"name": "coco", "color": "rose",
	})
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("friend color status = %d", response.StatusCode)
	}
	_ = response.Body.Close()

	response, err := client.Get(server.URL + "/api/bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var payload chat.Bootstrap
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if got := payload.FriendColors[chat.CocoName]; got != "rose" {
		t.Fatalf("friend color = %q, want rose", got)
	}
}

func TestGroupAPIAndOwnerPermission(t *testing.T) {
	server, aliceClient := newTestServer(t)
	defer server.Close()
	response := jsonRequest(t, aliceClient, http.MethodPost, server.URL+"/api/register", map[string]string{
		"name": "alice", "password": "password123",
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("alice register status = %d", response.StatusCode)
	}
	_ = response.Body.Close()
	bobJar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	bobClient := &http.Client{Jar: bobJar}
	response = jsonRequest(t, bobClient, http.MethodPost, server.URL+"/api/register", map[string]string{
		"name": "bob", "password": "password123",
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("bob register status = %d", response.StatusCode)
	}
	_ = response.Body.Close()

	response = jsonRequest(t, aliceClient, http.MethodPost, server.URL+"/api/groups", map[string]any{
		"name": "项目组", "signature": "一起推进", "members": []string{"bob"},
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create group status = %d", response.StatusCode)
	}
	var group chat.GroupView
	if err := json.NewDecoder(response.Body).Decode(&group); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if !group.IsOwner || group.Name != "项目组" {
		t.Fatalf("unexpected group response: %#v", group)
	}
	if !group.HistoryVisible {
		t.Fatal("new groups should allow history by default")
	}
	response = jsonRequest(t, aliceClient, http.MethodPatch, server.URL+"/api/groups", map[string]any{
		"id": group.ID, "name": "项目组", "historyVisible": false, "members": []string{"bob"},
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("disable history status = %d", response.StatusCode)
	}
	_ = response.Body.Close()

	response, err = bobClient.Get(server.URL + "/api/bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	var bobBootstrap chat.Bootstrap
	if err := json.NewDecoder(response.Body).Decode(&bobBootstrap); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if len(bobBootstrap.Groups) != 2 {
		t.Fatalf("bob groups = %#v", bobBootstrap.Groups)
	}
	response = jsonRequest(t, bobClient, http.MethodPatch, server.URL+"/api/groups", map[string]any{
		"id": group.ID, "name": "越权修改", "members": []string{"bob"},
	})
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("non-owner update status = %d", response.StatusCode)
	}
	_ = response.Body.Close()
}

func TestAttachmentAPIRequiresConfiguredStorage(t *testing.T) {
	server, client := newTestServer(t)
	defer server.Close()
	response := jsonRequest(t, client, http.MethodPost, server.URL+"/api/register", map[string]string{
		"name": "alice", "password": "password123",
	})
	_ = response.Body.Close()

	response, err := client.Get(server.URL + "/api/bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	var bootstrap chat.Bootstrap
	if err := json.NewDecoder(response.Body).Decode(&bootstrap); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if bootstrap.UploadsEnabled {
		t.Fatal("uploads unexpectedly enabled")
	}
	response = jsonRequest(t, client, http.MethodPost, server.URL+"/api/attachments/presign", map[string]any{
		"name": "test.txt", "contentType": "text/plain", "size": 12,
	})
	defer response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("presign status = %d", response.StatusCode)
	}
}

func TestAttachmentResponsePolicy(t *testing.T) {
	cases := []struct {
		contentType string
		download    bool
		disposition string
		ttl         time.Duration
	}{
		{"application/pdf", false, "inline", previewURLTTL},
		{"text/plain", false, "inline", previewURLTTL},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", false, "inline", previewURLTTL},
		{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", false, "inline", previewURLTTL},
		{"audio/flac", false, "inline", previewURLTTL},
		{"video/mp4", false, "inline", previewURLTTL},
		{"text/html", false, "attachment", attachmentURLTTL},
		{"image/svg+xml", false, "attachment", attachmentURLTTL},
		{"application/pdf", true, "attachment", previewURLTTL},
	}
	for _, testCase := range cases {
		disposition, ttl := attachmentResponsePolicy(testCase.contentType, testCase.download)
		if disposition != testCase.disposition || ttl != testCase.ttl {
			t.Errorf("attachmentResponsePolicy(%q, %v) = %q, %s", testCase.contentType,
				testCase.download, disposition, ttl)
		}
	}
}

func TestAttachmentPresignCompleteAndDelete(t *testing.T) {
	objects := newFakeObjectStore()
	server, client := newTestServerWithObjects(t, objects)
	defer server.Close()
	response := jsonRequest(t, client, http.MethodPost, server.URL+"/api/register", map[string]string{
		"name": "alice", "password": "password123",
	})
	_ = response.Body.Close()

	response = jsonRequest(t, client, http.MethodPost, server.URL+"/api/attachments/presign", map[string]any{
		"name": "notes.txt", "contentType": "text/plain", "size": 12,
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("presign status = %d", response.StatusCode)
	}
	var presigned struct {
		AttachmentID string            `json:"attachmentId"`
		UploadURL    string            `json:"uploadUrl"`
		Headers      map[string]string `json:"headers"`
	}
	if err := json.NewDecoder(response.Body).Decode(&presigned); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if presigned.AttachmentID == "" || presigned.UploadURL == "" || presigned.Headers["Content-Type"] != "text/plain" {
		t.Fatalf("presign response = %#v", presigned)
	}
	objects.metadata[objects.lastKey] = blob.ObjectMetadata{Size: 12, ContentType: "text/plain"}
	response = jsonRequest(t, client, http.MethodPost, server.URL+"/api/attachments/complete", map[string]string{
		"attachmentId": presigned.AttachmentID,
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("complete status = %d", response.StatusCode)
	}
	_ = response.Body.Close()

	request, err := http.NewRequest(http.MethodDelete, server.URL+"/api/attachments/"+presigned.AttachmentID, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", response.StatusCode)
	}
	if len(objects.deleted) != 1 || objects.deleted[0] != objects.lastKey {
		t.Fatalf("deleted keys = %#v", objects.deleted)
	}
}

func TestAttachmentCompleteRejectsMetadataMismatchAndCleansUp(t *testing.T) {
	objects := newFakeObjectStore()
	server, client := newTestServerWithObjects(t, objects)
	defer server.Close()
	response := jsonRequest(t, client, http.MethodPost, server.URL+"/api/register", map[string]string{
		"name": "alice", "password": "password123",
	})
	_ = response.Body.Close()
	response = jsonRequest(t, client, http.MethodPost, server.URL+"/api/attachments/presign", map[string]any{
		"name": "notes.txt", "contentType": "text/plain", "size": 12,
	})
	var presigned struct {
		AttachmentID string `json:"attachmentId"`
	}
	if err := json.NewDecoder(response.Body).Decode(&presigned); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	objects.metadata[objects.lastKey] = blob.ObjectMetadata{Size: 13, ContentType: "text/plain"}
	response = jsonRequest(t, client, http.MethodPost, server.URL+"/api/attachments/complete", map[string]string{
		"attachmentId": presigned.AttachmentID,
	})
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("complete status = %d", response.StatusCode)
	}
	if len(objects.deleted) != 1 || objects.deleted[0] == "" {
		t.Fatalf("mismatched object was not deleted: %#v", objects.deleted)
	}
}

func TestStickerUploadAndRemovalAPI(t *testing.T) {
	objects := newFakeObjectStore()
	server, client := newTestServerWithObjects(t, objects)
	defer server.Close()
	response := jsonRequest(t, client, http.MethodPost, server.URL+"/api/register", map[string]string{
		"name": "alice", "password": "password123",
	})
	_ = response.Body.Close()

	response = jsonRequest(t, client, http.MethodPost, server.URL+"/api/attachments/presign", map[string]any{
		"name": "sticker.png", "contentType": "image/png", "size": 128,
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("presign status = %d", response.StatusCode)
	}
	var presigned struct {
		AttachmentID string `json:"attachmentId"`
	}
	if err := json.NewDecoder(response.Body).Decode(&presigned); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	objects.metadata[objects.lastKey] = blob.ObjectMetadata{Size: 128, ContentType: "image/png"}
	response = jsonRequest(t, client, http.MethodPost, server.URL+"/api/attachments/complete", map[string]string{
		"attachmentId": presigned.AttachmentID,
	})
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("complete status = %d", response.StatusCode)
	}

	response = jsonRequest(t, client, http.MethodPost, server.URL+"/api/stickers", map[string]string{
		"attachmentId": presigned.AttachmentID,
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("add sticker status = %d", response.StatusCode)
	}
	var sticker chat.AttachmentView
	if err := json.NewDecoder(response.Body).Decode(&sticker); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if sticker.ID != presigned.AttachmentID || sticker.Kind != "image" || !sticker.Inline {
		t.Fatalf("sticker view = %#v", sticker)
	}

	response, err := client.Get(server.URL + "/api/bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	var bootstrap chat.Bootstrap
	if err := json.NewDecoder(response.Body).Decode(&bootstrap); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if len(bootstrap.Stickers) != 1 || bootstrap.Stickers[0].ID != sticker.ID {
		t.Fatalf("bootstrap stickers = %#v", bootstrap.Stickers)
	}

	request, err := http.NewRequest(http.MethodDelete, server.URL+"/api/stickers/"+sticker.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("remove sticker status = %d", response.StatusCode)
	}
}
